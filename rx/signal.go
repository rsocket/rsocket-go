package rx

import rs "github.com/jjeffcaii/reactor-go"

const (
	// SignalComplete indicated that subscriber was completed.
	SignalComplete = SignalType(rs.SignalTypeComplete)
	// SignalCancel indicates that subscriber was cancelled.
	SignalCancel = SignalType(rs.SignalTypeCancel)
	// SignalError indicates that subscriber has some faults.
	SignalError = SignalType(rs.SignalTypeError)
)

// SignalType is the signal of reactive events like `OnNext`, `OnComplete`, `OnCancel` and `OnError`.
type SignalType rs.SignalType

func (s SignalType) String() string {
	return rs.SignalType(s).String()
}
