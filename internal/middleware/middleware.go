package middleware

import (
	"fmt"
	"net/http"
	"time"
)

// TimerTrace замеряет время выполнения функции.
func TimerTrace(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		before := time.Now()
		next(w, r)
		fmt.Println(time.Since(before))
	}
}
