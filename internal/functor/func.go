package functor

import (
	"sync"

	"github.com/x-research-team/kernel/pkg/sys"
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
			r = append(r, sys.Call(f, e)...)
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
				r = append(r, sys.Call(f, e)...)
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
			result := sys.Call(f, e)
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
				result := sys.Call(f, e)
				if result[0].(bool) {
					r = append(r, e)
				}
			}(&wg, e)
		}
	}
	wg.Wait()
	return
}
