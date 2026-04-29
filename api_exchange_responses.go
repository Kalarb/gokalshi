package gokalshi

// Response types for exchange-related API endpoints.

// GetExchangeStatusResponse is the response from GET /exchange/status.
type GetExchangeStatusResponse struct {
	ExchangeActive              bool   `json:"exchange_active"`
	TradingActive               bool   `json:"trading_active"`
	ExchangeEstimatedResumeTime string `json:"exchange_estimated_resume_time"`
}

// AnnouncementResponse is a single exchange announcement.
type AnnouncementResponse struct {
	Type         AnnouncementType   `json:"type"`
	Message      string             `json:"message"`
	DeliveryTime string             `json:"delivery_time"`
	Status       AnnouncementStatus `json:"status"`
}

// GetExchangeAnnouncementsResponse is the response from GET /exchange/announcements.
type GetExchangeAnnouncementsResponse struct {
	Announcements []AnnouncementResponse `json:"announcements"`
}

// SeriesFeeChange is a single fee change record.
type SeriesFeeChange struct {
	ID            string  `json:"id"`
	SeriesTicker  string  `json:"series_ticker"`
	FeeType       FeeType `json:"fee_type"`
	FeeMultiplier float64 `json:"fee_multiplier"`
	ScheduledTs   string  `json:"scheduled_ts"`
}

// GetSeriesFeeChangesResponse is the response from GET /series/fee_changes.
type GetSeriesFeeChangesResponse struct {
	SeriesFeeChangeArr []SeriesFeeChange `json:"series_fee_change_arr"`
}

// TradingSession is a single open/close time window.
type TradingSession struct {
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
}

// StandardHoursWeek is the weekly trading schedule.
type StandardHoursWeek struct {
	StartTime string           `json:"start_time"`
	EndTime   string           `json:"end_time"`
	Monday    []TradingSession `json:"monday"`
	Tuesday   []TradingSession `json:"tuesday"`
	Wednesday []TradingSession `json:"wednesday"`
	Thursday  []TradingSession `json:"thursday"`
	Friday    []TradingSession `json:"friday"`
	Saturday  []TradingSession `json:"saturday"`
	Sunday    []TradingSession `json:"sunday"`
}

// MaintenanceWindow is a scheduled maintenance period.
type MaintenanceWindow struct {
	StartDatetime string `json:"start_datetime"`
	EndDatetime   string `json:"end_datetime"`
}

// ExchangeSchedule is the full exchange schedule.
type ExchangeSchedule struct {
	StandardHours      []StandardHoursWeek `json:"standard_hours"`
	MaintenanceWindows []MaintenanceWindow `json:"maintenance_windows"`
}

// GetExchangeScheduleResponse is the response from GET /exchange/schedule.
type GetExchangeScheduleResponse struct {
	Schedule ExchangeSchedule `json:"schedule"`
}

// GetUserDataTimestampResponse is the response from GET /exchange/user_data_timestamp.
type GetUserDataTimestampResponse struct {
	AsOfTime string `json:"as_of_time"`
}
