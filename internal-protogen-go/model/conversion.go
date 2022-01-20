package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func (ts *Timestamp) Scan(value interface{}) error {
	switch t := value.(type) {
	case time.Time:
		var err error
		ts.Timestamp = timestamppb.New(t)
		if err != nil {
			return err
		}
	default:
		return nil
	}
	return nil
}

func (t *Timestamp) Value() (driver.Value, error) {
	return t.Timestamp.AsTime(), nil
}

func (mv *ModelVersion) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal ModelVersion value:", value))
	}

	if err := json.Unmarshal(bytes, &mv); err != nil {
		return err
	}

	return nil
}

func (mv *ModelVersion) Value() (driver.Value, error) {
	valueString, err := json.Marshal(mv)
	return string(valueString), err
}
