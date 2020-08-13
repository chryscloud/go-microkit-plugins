package ai

type AnnotationList struct {
	Data []*Annotation `json:"data,omitempty"`
}

type Annotation struct {
	User              string        `json:"user,omitempty"`                     // required (owner of the account)
	DeviceName        string        `json:"device_name" binding:"required"`     // required: device name (required) identity of device
	RemoteStreamID    string        `json:"remote_stream_id,omitempty"`         // optional: if associated with storage, the ID of Chrysalis Cloud deviceID
	EventType         string        `json:"type" binding:"required"`            // required: event type: e.g. moving, exit, entry, stopped, parked, ...
	StartTimestamp    int64         `json:"start_timestamp" binding:"required"` // required: start of the event
	EndTimestamp      int64         `json:"end_timestamp,omitempty"`            // optional: event of the event
	ObjectType        string        `json:"object_type,omitempty"`              // optional: e.g. person, car, face, bag, roadsign,...
	ObjectID          string        `json:"object_id,omitempty"`                // optional: e.g. object id from the ML model
	ObjectTrackingID  string        `json:"object_tracking_id,omitempty"`       // optional: tracking id of the object
	ObjectCoordinate  *Coordinate   `json:"object_coordinate,omitempty"`        // optional: object coordinates within the image
	ObjectMask        []*Coordinate `json:"mask,omitempty"`                     // optional" object mask (polygon)
	ObjectSignature   []float64     `json:"object_signature,omitempty"`         // optional: signature of the detected item
	Confidence        float64       `json:"confidence,omitempty"`               // confidence of inference [0-1.0]
	ObjectBoundingBox *BoundingBox  `json:"object_bouding_box,omitempty"`       // optional: object bounding box
	Location          *Location     `json:"location,omitempty"`                 // optional: object GEO location
	MLModel           string        `json:"ml_model,omitempty"`                 // optional: description of the module that generated this event
	MLModelVersion    string        `json:"ml_model_version,omitempty"`         // optional: version of the ML model
	Width             int32         `json:"width,omitempty"`                    // optional: image width
	Height            int32         `json:"height,omitempty"`                   // optional: image height
	IsKeyframe        bool          `json:"is_keyrame,omitempty"`               // optional: true/false if this annotation is from keyframe
	VideoType         string        `json:"video_type,omitempty"`               // optional: e.g. mp4 filename, live stream, ...
	OffsetTimestamp   int64         `json:"offset_timestamp,omitempty"`         // optional: offset from the beginning
	OffsetDuration    int64         `json:"offset_duration,omitempty"`          // optional: duration from the offset
	OffsetFrameID     int64         `json:"offset_frame_id,omitempty"`          // optional: frame id of the offset
	OffsetPAcketID    int64         `json:"offset_packet_id,omitempty"`         // optional: offset of the packet
	// // extending the event message meta data (optional)
	CustomMeta1 string `json:"custom_meta_1,omitempty"` // e.g. gender, hair, car model, ...
	CustomMeta2 string `json:"custom_meta_2,omitempty"` // e.g. gender, hair, car model, ...
	CustomMeta3 string `json:"custom_meta_3,omitempty"` // e.g. gender, hair, car model, ...
	CustomMeta4 string `json:"custom_meta_4,omitempty"` // e.g. gender, hair, car model, ...
	CustomMeta5 string `json:"custom_meta_5,omitempty"` // e.g. gender, hair, car model, ...
}

type BoundingBox struct {
	Top    int32 `json:"top"`
	Left   int32 `json:"left"`
	Width  int32 `json:"width"`
	Height int32 `json:"height"`
}

type Coordinate struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}
