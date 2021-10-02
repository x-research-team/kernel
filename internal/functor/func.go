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
	"sync"

	"github.com/x-research-team/kernel/internal/dynamic"
)

func flatten(s []interface{}) (r []interface{}) {
	for _, e := range s {
		if i, ok := e.([]interface{}); ok {
			r = append(r, flatten(i)...)
		} else {
			r = append(r, e)
		}
	}
	return
}

func flattenAsync(s []interface{}) (r []interface{}) {
	var wg sync.WaitGroup
	for _, e := range s {
		if i, ok := e.([]interface{}); ok {
			r = append(r, flattenAsync(i)...)
		} else {
			wg.Add(1)
			go func(wg *sync.WaitGroup, e interface{}) {
				defer wg.Done()
				r = append(r, e)
			}(&wg, e)
		}
	}
	wg.Wait()
	return
}

func mapper(s []interface{}, f interface{}) (r []interface{}) {
	for _, e := range s {
		if i, ok := e.([]interface{}); ok {
			r = append(r, mapper(i, f)...)
		} else {
			r = append(r, dynamic.Call(f, e)...)
		}
	}
	return
}

func mapperAsync(s []interface{}, f interface{}) (r []interface{}) {
	var wg sync.WaitGroup
	for _, e := range s {
		if i, ok := e.([]interface{}); ok {
			r = append(r, mapperAsync(i, f)...)
		} else {
			wg.Add(1)
			go func(wg *sync.WaitGroup, e interface{}) {
				defer wg.Done()
				r = append(r, dynamic.Call(f, e)...)
			}(&wg, e)
		}
	}
	wg.Wait()
	return
}

func filter(s []interface{}, f interface{}) (r []interface{}) {
	for _, e := range s {
		if i, ok := e.([]interface{}); ok {
			r = append(r, filter(i, f)...)
		} else {
			result := dynamic.Call(f, e)
			if result[0].(bool) {
				r = append(r, e)
			}
		}
	}
	return
}

func filterAsync(s []interface{}, f interface{}) (r []interface{}) {
	var wg sync.WaitGroup
	for _, e := range s {
		if i, ok := e.([]interface{}); ok {
			r = append(r, filterAsync(i, f)...)
		} else {
			wg.Add(1)
			go func(wg *sync.WaitGroup, e interface{}) {
				defer wg.Done()
				result := dynamic.Call(f, e)
				if result[0].(bool) {
					r = append(r, e)
				}
			}(&wg, e)
		}
	}
	wg.Wait()
	return
}
