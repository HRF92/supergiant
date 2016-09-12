package model

type EntrypointListener struct {
	BaseModel

	// belongs_to Entrypoint
	Entrypoint   *Entrypoint `json:"entrypoint,omitempty"`
	EntrypointID *int64      `json:"entrypoint_id" gorm:"not null;index"`

	// EntrypointPort is the external port the user connects to
	EntrypointPort     int64  `json:"entrypoint_port" validate:"nonzero"`
	EntrypointProtocol string `json:"entrypoint_protocol" validate:"nonzero" g:"default=TCP"`

	// NodePort is the target port, what EntrypointPort maps to
	NodePort     int64  `json:"node_port" validate:"nonzero"`
	NodeProtocol string `json:"node_protocol" validate:"nonzero" sg:"default=TCP"`
}
