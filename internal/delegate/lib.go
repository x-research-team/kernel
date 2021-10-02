/*
 *   Copyright (c) 2021 Adel Urazov
 *   All rights reserved.

 *   Permission is hereby granted, free of charge, to any person obtaining a copy
 *   of this software and associated documentation files (the "Software"), to deal
 *   in the Software without restriction, including without limitation the rights
 *   to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *   copies of the Software, and to permit persons to whom the Software is
 *   furnished to do so, subject to the following conditions:
 
 *   The above copyright notice and this permission notice shall be included in all
 *   copies or substantial portions of the Software.
 
 *   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *   IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *   FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *   AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *   LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *   OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 *   SOFTWARE.
 */

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
