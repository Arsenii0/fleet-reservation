package kafkaevents

// Topic names shared between the reservation and fleet-deployer services.
const (
	ReserveResourceRequestTopic         = "reserve-resource-request"
	ReleaseInstancesRequestTopic        = "release-instances-request"
	UpdateReservationInstanceStateTopic = "update-reservation-instance-state-request"
)
