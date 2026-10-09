package bot

import (
	"context"
	"errors"
	"sync"

	"github.com/AndriiPerehinets/TgBOT/internal/telegram"
	"github.com/AndriiPerehinets/TgBOT/internal/utils"
)

type pool struct {
	bot      *Bot
	taskChan []chan telegram.Update
	wg       sync.WaitGroup
}

func NewPool(bot *Bot, maxGoroutines, chBufSize int) *pool {
	ch := make([]chan telegram.Update, maxGoroutines)

	for i := 0; i < maxGoroutines; i++ {
		ch[i] = make(chan telegram.Update, chBufSize)
	}
	return &pool{
		bot:      bot,
		taskChan: ch,
	}
}

func (p *pool) CreateWorkerPool(ctx context.Context, maxGoroutines int) {
	for i := 0; i < maxGoroutines; i++ {
		p.wg.Add(1)
		go p.newWorker(p.taskChan[i])
	}
}

func (p *pool) newWorker(ch chan telegram.Update) {
	defer p.wg.Done()

	for {
		for U := range ch {
			p.bot.Logger.Printf("Response data:\t %#v\n\n", U)

			err := p.bot.fetchMessage(&U.Message)
			if err != nil {
				if !errors.Is(err, utils.ErrUserNotified) {
					err = errors.Join(err, p.bot.sendText(&U.Message, "Sorry, an error occured"))
				}
				p.bot.Logger.Println(err)
			}
		}
	}
}
