package bkmicro

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/punky97/go-codebase/core/drivers/bkredis"
	"github.com/punky97/go-codebase/core/drivers/cassandra"
	"github.com/punky97/go-codebase/core/drivers/es"
	"github.com/punky97/go-codebase/core/drivers/mongo"
	"github.com/punky97/go-codebase/core/drivers/mongo/mgo_mock"
	"github.com/punky97/go-codebase/core/drivers/mysql"
	"github.com/punky97/go-codebase/core/drivers/queue"
	"github.com/punky97/go-codebase/core/logger"
)

// CreateMysqlConnection --
func (app *App) CreateMysqlConnection(conn mysql.Connection) *mysql.BkMysql {
	if conn == nil {
		conn = mysql.DefaultMysqlConnectionFromConfig()
	}
	db, err := mysql.NewConnection(conn)
	if err != nil {
		logger.BkLog.Panic(err)
	}
	return db
}

// CreateMysqlConnection --
func CreateMysqlMock() (*mysql.BkMysql, sqlmock.Sqlmock) {
	db, sqlMock, err := sqlmock.New()
	db.Begin()
	if err != nil {
		logger.BkLog.Panic(err)
	}

	sqlDb, _ := mysql.NewBkMysqlFromRawDB(db)
	return sqlDb, sqlMock
}

// CreatePostgreConnection --
func (app *App) CreatePostgreConnection(conn mysql.Connection) *mysql.BkMysql {
	if conn == nil {
		conn = mysql.DefaultPostGreConnectionFromConfig()
	}
	db, err := mysql.NewPostgreConnection(conn)
	if err != nil {
		logger.BkLog.Panic(err)
	}
	return db
}

// CreateEsConnection --
func (app *App) CreateEsConnection(conn *es.Connection) *es.BkEs {
	if conn == nil {
		conn = es.DefaultEsConnectionFromConfig()
	}
	db, err := es.NewConnection(conn)
	if err != nil {
		logger.BkLog.Panic(err)
	}
	return db
}

// CreateMongoConnection --
func (app *App) CreateMongoConnection(conn *mongo.Connection) *mongo.BkMongo {
	if conn == nil {
		conn = mongo.DefaultMongoConnectionFromConfig()
	}
	db, err := mongo.NewConnection(conn)
	if err != nil {
		logger.BkLog.Panic(err)
	}
	return db
}

// CreateMongoConnection --
func CreateMongoMock() (*mongo.BkMongo, *mgo_mock.MgoMock) {
	session := &mgo_mock.Session{}
	bkMongo := &mongo.BkMongo{
		Session: session,
	}
	return bkMongo, &mgo_mock.MgoMock{Session: session}
}

// CreateCassandraConnection --
func (app *App) CreateCassandraConnection(conn *cassandra.Connection) *cassandra.BkCassandra {
	if conn == nil {
		conn = cassandra.DefaultCasssandraConnectionFromConfig()
	}
	db, err := cassandra.NewConnection(conn)
	if err != nil {
		logger.BkLog.Panic(err)
	}
	return db
}

func (app *App) CreateRedisConnectionWithLayer(ctx context.Context, conn bkredis.Connection, layer string) *bkredis.RedisClient {
	switch layer {
	case BatchLayer:
		if conn == nil {
			conn = bkredis.RedisBatchLayerConnection()
		}
	case ServeLayer:
		if conn == nil {
			conn = bkredis.RedisServeLayerConnection()
		}
	default:
		if conn == nil {
			conn = bkredis.DefaultRedisConnectionFromConfig()
		}
	}

	db, err := bkredis.NewConnection(ctx, conn)
	if err != nil {
		logger.BkLog.Panic(err)
	}
	return db
}

// CreateRedisConnection --
func (app *App) CreateRedisConnection(ctx context.Context, conn bkredis.Connection) *bkredis.RedisClient {
	if conn == nil {
		conn = bkredis.DefaultRedisConnectionFromConfig()
	}
	db, err := bkredis.NewConnection(ctx, conn)
	if err != nil {
		logger.BkLog.Panic(err)
	}
	return db
}

// CreateRabbitMQProducerConnection creates new producer connection
func (app *App) CreateRabbitMQProducerConnection(conn *queue.RabbitMqConfiguration) *queue.Producer {
	internalQueueSize := 1000
	retries := 10
	numThread := 1
	if conn == nil {
		misc := queue.LoadOtherConfigParams()
		exch := queue.LoadExchConfig()
		conn = &queue.RabbitMqConfiguration{URI: misc["uri"].(string), ExchangeConfig: *exch}
		internalQueueSize = misc["internal_queue_size"].(int)
		retries = misc["retries"].(int)
		numThread, _ = misc["num_thread"].(int)

		if numThread <= 0 {
			numThread = 1
		}
	}

	p := queue.NewRMQProducerFromConf(conn, internalQueueSize, retries)

	if err := p.ConnectMulti(numThread); err != nil {
		logger.BkLog.Panic(err)
	}
	p.Start()

	return p
}
