package process

import (
	"github.com/AndriiPerehinets/TgBOT/internal/telegram"
)

func CreateWorkerPool(PoolSize int, ch chan telegram.Update) {
	for i := 0; i < PoolSize; i++ {

	}
}

func Worker(U telegram.Update) {

}
