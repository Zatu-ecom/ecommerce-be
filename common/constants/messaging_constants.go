package constants

const (
	// Messaging defaults
	DEFAULT_COMMANDS_EXCHANGE = "ecom.commands"
	DEFAULT_EVENTS_EXCHANGE   = "ecom.events"
	DEFAULT_EXCHANGE_TYPE     = "topic"

	// Message headers
	MSG_HEADER_RETRY_COUNT = "x-retry-count"
	MSG_HEADER_TENANT_ID   = "x-tenant-id"
	MSG_HEADER_ACTOR_ID    = "x-actor-id"
	MSG_HEADER_TRACE_ID    = "x-trace-id"
)
