package statistic

import "time"

const (
	StatisticSaleSourceTotal     = "total"
	StatisticSaleFieldTotalSales = "total_sales"

	HourLayout        = "2006-01-02T15:00:00"
	DayLayout         = "2006-01-02T00:00:00"
	ResponseDayLayout = "2006-01-02"
	ThreeDays         = 3 * 24 * time.Hour
	ThreeMonths       = 90 * 24 * time.Hour

	TypeHour  = "hour"
	TypeDay   = "day"
	TypeWeek  = "week"
	TypeMonth = "month"

	EventAddToCart             = "add_to_cart"
	ViewContent                = "view_content"
	ReachedCheckout            = "reached_checkout"
	EntityReachedCheckout      = "entity_reached_checkout"
	Purchase                   = "purchase"
	EventSearch                = "search"
	EventCheckoutButtonClicked = "checkout_button_clicked"
	EventCartDrawerIconClicked = "cart_drawer_icon_clicked"
	EventBuyNowButtonClicked   = "buy_now_button_clicked"
	EventInitiateCheckout      = "initiate_checkout"
	EventUseCouponCode         = "use_coupon_code"
	EventAddShippingInfo       = "add_shipping_info"
	EventSelectShippingMethod  = "select_shipping_method"
	EventAddPaymentInfo        = "add_payment_info"

	FbEventSearch                = "Search"
	FbEventViewContent           = "ViewContent"
	FbEventCheckoutButtonClicked = "CheckoutButtonClicked"
	FbEventCartDrawerIconClicked = "CartDrawerIconClicked"
	FbEventBuyNowButtonClicked   = "BuyNowButtonClicked"
	FbEventAddToCart             = "AddToCart"
	FbEventInitiateCheckout      = "InitiateCheckout"
	FbEventUseCouponCode         = "UseCouponCode"
	FbEventAddShippingInfo       = "AddShippingInfo"
	FbEventSelectShippingMethod  = "SelectShippingMethod"
	FbEventAddPaymentInfo        = "AddPaymentInfo"
	FbEventPurchase              = "Purchase"

	HasSaleNo   = "no"
	HasSaleYes  = "yes"
	HasSaleBoth = "both"

	Version1 = "v1"

	DefaultCountLimit = 500
	LimitExportInAPI  = 2000
	LimitRowPerPage   = 2000

	ExportTypeConversionRate = "conversion_rate"
	ExportTypeSaleReportV2   = "sale_report_v2"
	ExportTypeSaleReportV3   = "sale_report_v3"
)

var (
	SaleReportV2IgnoreFields = []string{
		"staff_name",
	}

	AllowedFacebookEvents = []string{
		EventSearch,
		ViewContent,
		EventCheckoutButtonClicked,
		EventCartDrawerIconClicked,
		EventBuyNowButtonClicked,
		EventAddToCart,
		EventInitiateCheckout,
		EventUseCouponCode,
		EventAddShippingInfo,
		EventSelectShippingMethod,
		EventAddPaymentInfo,
		Purchase,
	}

	SbaseEventToFacebookEvent = map[string]string{
		EventSearch:                FbEventSearch,
		ViewContent:                FbEventViewContent,
		EventCheckoutButtonClicked: FbEventCheckoutButtonClicked,
		EventCartDrawerIconClicked: FbEventCartDrawerIconClicked,
		EventBuyNowButtonClicked:   FbEventBuyNowButtonClicked,
		EventAddToCart:             FbEventAddToCart,
		EventInitiateCheckout:      FbEventInitiateCheckout,
		EventUseCouponCode:         FbEventUseCouponCode,
		EventAddShippingInfo:       FbEventAddShippingInfo,
		EventSelectShippingMethod:  FbEventSelectShippingMethod,
		EventAddPaymentInfo:        FbEventAddPaymentInfo,
		Purchase:                   FbEventPurchase,
	}
)
