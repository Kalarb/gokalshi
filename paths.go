package gokalshi

// API path constants. Each domain file's methods use these to build endpoint URLs.
const (
	pathAccount           = "/trade-api/v2/account"
	pathAPIKeys           = "/trade-api/v2/api_keys"
	pathCommunications    = "/trade-api/v2/communications"
	pathEvents            = "/trade-api/v2/events"
	pathExchange          = "/trade-api/v2/exchange"
	pathHistorical        = "/trade-api/v2/historical"
	pathIncentivePrograms = "/trade-api/v2/incentive_programs"
	pathLiveData          = "/trade-api/v2/live_data"
	pathMarkets           = "/trade-api/v2/markets"
	pathMilestones        = "/trade-api/v2/milestones"
	pathMVECollections    = "/trade-api/v2/multivariate_event_collections"
	pathPortfolio         = "/trade-api/v2/portfolio"
	pathSearch            = "/trade-api/v2/search"
	pathSeries            = "/trade-api/v2/series"
	pathStructuredTargets = "/trade-api/v2/structured_targets"

	// Sub-paths under pathPortfolio.
	pathOrders      = pathPortfolio + "/orders"
	pathEventOrders = pathPortfolio + "/events/orders"
	pathOrderGroups = pathPortfolio + "/order_groups"
	pathSubaccounts = pathPortfolio + "/subaccounts"
)
