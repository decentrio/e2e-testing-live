package cosmos

import (
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
		println("check attrKey: ", attrKey)
		for _, attr := range event.Attributes {
			println("check attrKey: ", attr.Key)
			if string(attr.Key) == attrKey {
				println("check result: ", string(attr.Value))
				return string(attr.Value), true
			}
		}
	}
	return "", false
}
