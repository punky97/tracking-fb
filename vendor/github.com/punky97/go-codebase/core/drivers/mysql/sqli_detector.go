package mysql

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/punky97/go-codebase/core/bkgrpc/meta"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/spf13/viper"
	"github.com/xwb1989/sqlparser"
)

var ErrUnknownStatementType = errors.New("sqli: unknown statement type")
var ErrPreventSQLInjection = errors.New("sqli: prevent execute query due to suspect sql injection")
var ErrUsingStringBuilder = errors.New("sqli: current query might using string builder")

var (
	defaultSuspiciousSubstrings = []string{
		"information_schema",
		"@@version",
		"char(",
		"chr(",
		"load_file(",
		"sleep(",
		"hex(",
		"--",
		"/*",
		"*/",
		"0x",
		";",
	}
)

var scanner *sqliScanner

type sqliScanner struct {
	enableBlockSuspicious bool
	whitelistSuspicious   []string
	suspiciousSubstrings  []string

	enableBlockStringBuilder bool
	whitelistStringBuilder   []string
	stringBuilderWhere       bool
	stringBuilderOrderBy     bool
	stringBuilderGroupBy     bool
	stringBuilderHaving      bool
}

// initSqliScanner --
func initSqliScanner() {
	suspiciousSubstrings := viper.GetStringSlice("security.sqli.suspicious_substrings")
	if len(suspiciousSubstrings) == 0 {
		suspiciousSubstrings = defaultSuspiciousSubstrings
	}
	sort.Slice(suspiciousSubstrings, func(i, j int) bool {
		return len(suspiciousSubstrings[i]) > len(suspiciousSubstrings[j])
	})

	scanner = &sqliScanner{
		enableBlockSuspicious: viper.GetBool("security.sqli.enable_block_suspicious"),
		whitelistSuspicious:   viper.GetStringSlice("security.sqli.whitelist_suspicious"),
		suspiciousSubstrings:  suspiciousSubstrings,

		enableBlockStringBuilder: viper.GetBool("security.sqli.enable_block_string_builder"),
		whitelistStringBuilder:   viper.GetStringSlice("security.sqli.whitelist_string_builder"),
		stringBuilderWhere:       viper.GetBool("security.sqli.string_builder_where"),
		stringBuilderOrderBy:     viper.GetBool("security.sqli.string_builder_order_by"),
		stringBuilderGroupBy:     viper.GetBool("security.sqli.string_builder_group_by"),
		stringBuilderHaving:      viper.GetBool("security.sqli.string_builder_having"),
	}
}

func (s *sqliScanner) Scan(ctx context.Context, q string, lenArgs int) (err error) {
	var suspicious string
	defer func() {
		switch err {
		case ErrUsingStringBuilder:
			if !s.enableBlockStringBuilder {
				err = nil
			}

			callStack := debug.Stack()

			for _, wm := range s.whitelistStringBuilder {
				if strings.Contains(string(callStack), wm) {
					err = nil
					return
				}
			}

			logger.BkLog.Infow("[sqli] Query might using string builder",
				"block", s.enableBlockStringBuilder,
				"query", q,
				"method", meta.GetDMSMethodName(ctx),
				"source_pod_name", meta.GetDMSSourcePodName(ctx),
				"source_ip", meta.GetDMSSourceIP(ctx),
			)
		case ErrPreventSQLInjection:
			if !s.enableBlockSuspicious {
				err = nil
			}

			callStack := debug.Stack()

			for _, wm := range s.whitelistSuspicious {
				if strings.Contains(string(callStack), wm) {
					err = nil
					return
				}
			}

			logger.BkLog.Infow("[sqli] Query contains suspicious character",
				"block", s.enableBlockSuspicious,
				"query", q,
				"suspicious_substring", suspicious,
				"method", meta.GetDMSMethodName(ctx),
				"source_pod_name", meta.GetDMSSourcePodName(ctx),
				"source_ip", meta.GetDMSSourceIP(ctx),
			)
		default:
		}
	}()

	// contain suspicious
	lq := strings.ToLower(q)
	for _, s := range s.suspiciousSubstrings {
		if strings.Contains(lq, s) {
			suspicious = s
			break
		}
	}

	if len(suspicious) > 0 {
		return ErrPreventSQLInjection
	}

	// using string builder?
	if s.stringBuilderWhere && strings.Contains(lq, "where") && lenArgs == 0 {
		return ErrUsingStringBuilder
	}

	if s.stringBuilderGroupBy && strings.Contains(lq, "group by") {
		return ErrUsingStringBuilder
	}

	if s.stringBuilderOrderBy && strings.Contains(lq, "order by") {
		return ErrUsingStringBuilder
	}

	if s.stringBuilderHaving && strings.Contains(lq, "having") {
		return ErrUsingStringBuilder
	}

	return
}

func queryAnalysis(q string) (err error) {
	// advance parser
	var stmt sqlparser.Statement
	stmt, err = sqlparser.Parse(q)
	_, err = sqlparser.Parse(q)
	if err != nil {
		return
	}

	//Otherwise do something with stmt
	switch stmt := stmt.(type) {
	case *sqlparser.Select:
		return analysisSelectStmt(stmt)
	case *sqlparser.Insert:
		return analysisInsertStmt(stmt)
	case *sqlparser.Update:
		return analysisUpdateStmt(stmt)
	case *sqlparser.Delete:
		return analysisDeleteStmt(stmt)
	default:
		err = ErrUnknownStatementType
	}

	return
}

func analysisSelectStmt(stmt *sqlparser.Select) (err error) {
	fmt.Printf("cache: %v \n", stmt.Cache)
	fmt.Printf("comments: %v \n", sqlparser.String(stmt.Comments))
	fmt.Printf("distinct: %v \n", stmt.Distinct)
	fmt.Printf("hints: %v \n", stmt.Hints)
	fmt.Printf("select fields: %v \n", sqlparser.String(stmt.SelectExprs))
	fmt.Printf("from: %v \n", sqlparser.String(stmt.From))
	fmt.Printf("where: %v \n", sqlparser.String(stmt.Where))
	fmt.Printf("group by: %v \n", sqlparser.String(stmt.GroupBy))
	fmt.Printf("having: %v \n", sqlparser.String(stmt.Having))
	fmt.Printf("order by: %v \n", sqlparser.String(stmt.OrderBy))
	fmt.Printf("limit: %v \n", sqlparser.String(stmt.Limit))
	fmt.Printf("lock: %v \n", stmt.Lock)

	return
}

func analysisInsertStmt(stmt *sqlparser.Insert) (err error) {
	fmt.Printf("action: %v \n", stmt.Action)
	fmt.Printf("comments: %v \n", sqlparser.String(stmt.Comments))
	fmt.Printf("ignore: %v \n", stmt.Ignore)
	fmt.Printf("table: %v \n", sqlparser.String(stmt.Table))
	fmt.Printf("partitions: %v \n", sqlparser.String(stmt.Partitions))
	fmt.Printf("columns: %v \n", sqlparser.String(stmt.Columns))
	fmt.Printf("rows: %v \n", sqlparser.String(stmt.Rows))
	fmt.Printf("on dup: %v \n", sqlparser.String(stmt.OnDup))

	return
}

func analysisUpdateStmt(stmt *sqlparser.Update) (err error) {
	fmt.Printf("comments: %v \n", sqlparser.String(stmt.Comments))
	fmt.Printf("table exprs: %v \n", sqlparser.String(stmt.TableExprs))
	fmt.Printf("exprs: %v \n", sqlparser.String(stmt.Exprs))
	fmt.Printf("where: %v \n", sqlparser.String(stmt.Where))
	fmt.Printf("order by: %v \n", sqlparser.String(stmt.OrderBy))
	fmt.Printf("limit: %v \n", sqlparser.String(stmt.Limit))

	return
}

func analysisDeleteStmt(stmt *sqlparser.Delete) (err error) {
	fmt.Printf("comments: %v \n", sqlparser.String(stmt.Comments))
	fmt.Printf("targets: %v \n", sqlparser.String(stmt.Targets))
	fmt.Printf("tableExprs: %v \n", sqlparser.String(stmt.TableExprs))
	fmt.Printf("partitions: %v \n", sqlparser.String(stmt.Partitions))
	fmt.Printf("where: %v \n", sqlparser.String(stmt.Where))
	fmt.Printf("orderBy: %v \n", sqlparser.String(stmt.OrderBy))
	fmt.Printf("limit: %v \n", sqlparser.String(stmt.Limit))

	return
}

// ScanSQLInjection --
func ScanSQLInjection(ctx context.Context, q string, lenArgs int) (err error) {
	if scanner == nil {
		initSqliScanner()
	}

	if ctx == nil {
		ctx = context.Background()
	}

	return scanner.Scan(ctx, q, lenArgs)
}
