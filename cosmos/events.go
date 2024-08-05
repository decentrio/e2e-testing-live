package cosmos

import (
	"encoding/base64"
	abcitypes "github.com/cometbft/cometbft/abci/types"
)

// AttributeValue returns an event attribute value given the eventType and attribute key tuple.
// In the event of duplicate types and keys, returns the first attribute value found.
// If not found, returns empty string and false.
func AttributeValue(events []abcitypes.Event, eventType, attrKey string) (string, bool) {
	for _, event := range events {
		if event.Type != eventType {
			continue
		}
		for _, attr := range event.Attributes {
			key, err := base64.StdEncoding.DecodeString(attr.Key)
			if err != nil {
				return "", false
			}
			if string(key) == attrKey {
				value, err := base64.StdEncoding.DecodeString(attr.Value)
				if err != nil {
					return "", false
				}
				return string(value), true
			}
		}
	}

	return "", false
}
