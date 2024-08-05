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
			key, _ := base64.StdEncoding.DecodeString(attr.Key)
			println("check key: ", string(key))
			if string(key) == attrKey {
				value, _ := base64.StdEncoding.DecodeString(attr.Value)
				return string(value), true
			}
		}
	}
	return "", false
}
