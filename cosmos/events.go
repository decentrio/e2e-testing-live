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
		println("check event: ", event.String())
		for _, attr := range event.Attributes {
			println("check attrKey: ", attrKey)
			println("check Key: ", attr.Key)
			println("check value: ", attr.Value)
			if string(attr.Key) == attrKey {
				return string(attr.Value), true
			}
		}
	}
	return "", false
}
