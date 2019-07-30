package bus

import "reflect"

type channelEventHandler struct {
    callBackFunction    reflect.Value
    runOnce             bool
}
