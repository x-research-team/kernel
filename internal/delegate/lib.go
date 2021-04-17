package delegate

import (
	"log"
	"sync"
)

type TItem interface{}
type TQueue []TItem

func Queue() TQueue {
	return make(TQueue, 0)
}

func (s *TQueue) Add(x ...TItem) {
	for _, i := range x {
		if i == nil {
			continue
		}
		*s = append(*s, i)
	}
}

type TFuncDelegate func(TItem) error

func (s *TQueue) Commit(delegates ...TFuncDelegate) {
	defer s.reset()
	var wg sync.WaitGroup
	for cred := range s.wait() {
		wg.Add(len(delegates))
		for _, delegate := range delegates {
			go func(wg *sync.WaitGroup, delegate TFuncDelegate) {
				defer wg.Done()
				if err := delegate(cred); err != nil {
					log.Printf("%v\n", err)
				}
			}(&wg, delegate)
		}
		wg.Wait()
	}
}

func (s TQueue) wait() <-chan TItem {
	ch := make(chan TItem)
	go func(list TQueue) {
		defer close(ch)
		for _, i := range list {
			select {
			case ch <- i:
			default:
				continue
			}
		}
	}(s)
	return ch
}

func (s *TQueue) reset() {
	*s = Queue()
}
