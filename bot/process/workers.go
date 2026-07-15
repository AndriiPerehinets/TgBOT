package process

import (
	"sv/types"
)

func CreateWorkerPool(PoolSize int, ch chan types.Update) {
	for i := 0; i < PoolSize; i++ {

	}
}

func Worker(U types.Update) {

}
