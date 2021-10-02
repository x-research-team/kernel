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

package functor

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/x-research-team/kernel/internal/dynamic"
)

// Functor Функтор
type Functor struct {
	errors     []error
	history    []*Functor
	branches   []*Functor
	collection []interface{}
}

// FunctionMode Тип исполнения
type FunctionMode int

const (
	// Async Асинхронное исполнение (Experimental)
	Async FunctionMode = iota
	// Sync Синхронное исполнение
	Sync
)

// F (args: ...any) Создать функтор
func F(args ...interface{}) *Functor {
	return &Functor{collection: args}
}

// Branch (branch: func (F): F) Создать ветку исполенения
func (functor *Functor) Branch(branch func(*Functor) *Functor) *Functor {
	functor.branches = append(functor.branches, branch(F(functor.collection...)))
	return functor
}

func (functor *Functor) backup() {
	c := F(functor.collection...)
	c.errors = functor.errors
	c.history = functor.history
	functor.history = append(functor.history, c)
}

func (functor *Functor) Map(mode FunctionMode, lambdas ...dynamic.Lambda) *Functor {
	functor.backup()
	switch mode {
	case Async:
		for _, lambda := range lambdas {
			functor.collection = mapperAsync(functor.collection, lambda)
		}
	case Sync:
		for _, lambda := range lambdas {
			functor.collection = mapper(functor.collection, lambda)
		}
	}
	return functor
}

func (functor *Functor) Filter(mode FunctionMode, lambdas ...dynamic.Lambda) *Functor {
	functor.backup()
	switch mode {
	case Async:
		for _, lambda := range lambdas {
			functor.collection = filterAsync(functor.collection, lambda)
		}
	case Sync:
		for _, lambda := range lambdas {
			functor.collection = filter(functor.collection, lambda)
		}
	}
	return functor
}

// Apply Применить обработчик к данным
func (functor *Functor) Apply(mode FunctionMode, lambdas ...dynamic.Lambda) *Functor {
	functor.backup()
	switch mode {
	case Async:
		var (
			wg sync.WaitGroup
			m  sync.RWMutex
		)
		for _, lambda := range lambdas {
			wg.Add(1)
			go func(wg *sync.WaitGroup, lambda dynamic.Lambda) {
				defer wg.Done()
				m.Lock()
				functor.collection = dynamic.Call(lambda, functor.collection)
				m.Unlock()
			}(&wg, lambda)
		}
		wg.Wait()
	case Sync:
		for _, lambda := range lambdas {
			functor.collection = dynamic.Call(lambda, functor.collection...)
		}
	}

	return functor
}

// Pipe (f: func (...any): []any) Выполнить команду в ветке исполнения
func (functor *Functor) Pipe(f ...interface{}) (*Functor, error) {
	var returns []interface{}
	for _, o := range f {
		results := dynamic.Call(o, functor.collection...)
		for i := range results {
			switch results[i].(type) {
			case error:
				functor.errors = append(functor.errors, results[i].(error))
				results = remove(results, i)
			case nil:
				results = remove(results, i)
			}
		}
		returns = append(returns, results...)
	}
	c := F(returns...)
	if len(functor.errors) != 0 {
		var errs []string
		for _, err := range functor.errors {
			errs = append(errs, err.Error())
		}
		return c, fmt.Errorf("[ERR] %v", strings.Join(errs, ", "))
	}
	return c, nil
}

func (functor *Functor) Result() *FunctorResult {
	branches := make([]interface{}, 0, 0)
	for _, branch := range functor.branches {
		branches = append(branches, branch.Result())
	}
	return &FunctorResult{
		Root:     functor.collection,
		Branches: branches,
	}
}

type FunctorResult struct {
	Root     []interface{}
	Branches []interface{}
}

func (r *FunctorResult) ToJson() json.RawMessage {
	buffer, _ := json.Marshal(r)
	return buffer
}

func remove(s []interface{}, i int) []interface{} {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
