package handler

import(
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/repository"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/models"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/service"
	"github.com/go-chi/chi/v5"
	"net/http"
	"github.com/rs/zerolog/log"
	"time"
	"encoding/json"
	"errors"
	"context"
	"strings"
	"io"
	"runtime/debug"
	"net/http/httputil"
)

const TOKEN_EXP = time.Hour * 24

type ctxKey string

const(
	userIDKey ctxKey = "userID"
)

type claims struct {
	jwt.RegisteredClaims
	UserID int
}

type responseData struct {
	status int
	size int
}
type responseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func (wr *responseWriter) Write(b []byte) (int, error) {
	size, err := wr.ResponseWriter.Write(b)
	wr.responseData.size += size
	return size, err
}

func (wr *responseWriter) WriteHeader(statusCode int) {
	wr.ResponseWriter.WriteHeader(statusCode)
	wr.responseData.status = statusCode
}

type usersController struct {
	repo *repository.Repository
	sk any
}

// Создание нового контроллера для работы с пользовательским АПИ
func NewUsersController(repo *repository.Repository, sk any) *usersController {
	return &usersController{repo: repo, sk: sk}
}

func (c *usersController) ApplyTo(mux chi.Router) {
	mux.Use()

	mux.Route("/api/user", func(r chi.Router) {
		r.Use(recoveryPanic, c.logRequest)
		r.Post("/login", c.login)
		r.With(checkJSON).Post("/register", c.register)
		r.With(c.checkAuthorization).Route("/orders", func(r chi.Router) {
			r.Post("/", c.postOrders)
			r.Get("/", c.getOrders)
		})
		r.With(c.checkAuthorization).Route("/balance", func(r chi.Router) {
			r.Get("/", c.balance)
			r.Post("/withdraw", c.withdraw)
		})
		r.With(c.checkAuthorization).Get("/withdrawals", c.withdrawals)
	})
	
}

func recoveryPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					log.Debug().Err(err).Str("stack", string(debug.Stack())).Msg("panic")
				} else {
					log.Debug().Any("recovered", r).Msg("panic")
				}
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
func (c *usersController) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqDump, err := httputil.DumpRequest(r, true)

		if err != nil {
			log.Debug().
				Err(err).
				Msg("dump request error")
		}

		log.Info().Str("req_dump", string(reqDump)).Msg("")
		responseD := &responseData{status: http.StatusOK}
		lw := responseWriter{ResponseWriter: w, responseData: responseD}

		next.ServeHTTP(&lw, r)

		log.Info().
			Str("request_method", r.Method).
			Str("request_uri", r.RequestURI).
			Str("request_content_type", r.Header.Get("Content-Type")).
			Int("response_status", responseD.status).
			Int("response_size", responseD.size).
			Str("response_content_type", lw.Header().Get("Content-Type")).
			Msg("")
	})
}

func (c *usersController) checkAuthorization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authToken, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")

		if !found {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		userID := getUserID(authToken, c.sk)
		if userID < 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		exist, err := c.repo.Users.ExistByID(r.Context(), userID)

		if err != nil {
			log.Debug().Err(err).Msg("error while checking user")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !exist {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getUserID(tokenString string, sk any) int {
	claims := &claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return sk, nil
	})

	if err != nil {
		log.Debug().Err(err).Msgf("parse jwt error: %v", err)
		return -1
	}

	if !token.Valid {
		log.Debug().Interface("token", token).Msg("Token is not valid")
		return -1
	}

	return claims.UserID
}

func checkPlain(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(strings.HasPrefix(r.Header.Get("Content-Type"), "text/plain") || (r.Header.Get("Content-Type") == "")) {
			w.Header().Set("Content-Type", "text/plain;")
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func checkJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			w.Header().Set("Content-Type", "text/plain;")
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func buildJWTString(userID int, sk any) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims {
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString(sk)
	if err != nil {
		log.Debug().Err(err).Msg("jwt error")
		return "", fmt.Errorf("can't sign token: %w", err)
	}

	return tokenString, nil
}

func (c *usersController) login(w http.ResponseWriter, r *http.Request) {
	var u models.User
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&u); err != nil {
		log.Debug().Err(err).Msgf("can't parse json: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	uID, err := c.repo.Users.Auth(r.Context(), u)

	if err != nil {
		if errors.Is(err, repository.ErrWrongCredentials) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	token, err := buildJWTString(uID, c.sk)
	if err != nil {
		log.Debug().Err(err).Msg("can't build jwt")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer " + token)
	w.Write([]byte("Success"))
}


func (c *usersController) register(w http.ResponseWriter, r *http.Request) {
	var u models.User
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&u); err != nil {
		log.Debug().Err(err).Msgf("can't parse json: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	uID, err := c.repo.Users.Register(r.Context(), u)

	if err != nil {
		if errors.Is(err, repository.ErrLoginExist) {
			w.WriteHeader(http.StatusConflict)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	token, err := buildJWTString(uID, c.sk)
	if err != nil {
		log.Debug().Err(err).Msg("can't build jwt")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer " + token)
	w.Write([]byte("Success"))
}

func (c *usersController) postOrders(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(int)
	var order models.Order

	bytes, err := io.ReadAll(io.LimitReader(r.Body, 100))

	if err != nil {
		log.Debug().Err(err).Msgf("can't read from body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = service.CheckOrderNum(string(bytes))

	if err != nil {
		if errors.Is(err, service.ErrFormatOrderNum) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		log.Debug().Err(err).Msgf("can't parse number: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	order.Number = string(bytes)
	order.UID = &userID

	err = c.repo.Orders.Create(r.Context(), order)
	if err != nil {
		if errors.Is(err, repository.ErrOrderExist) {
			w.WriteHeader(http.StatusOK)
			return
		}

		if errors.Is(err, repository.ErrOrderLoadedByOtherUser) {
			w.WriteHeader(http.StatusConflict)
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

func (c *usersController) getOrders(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(int)
	orders, err := c.repo.Orders.List(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	body, err := json.Marshal(orders)

	if err != nil {
		log.Debug().Err(err).Msg("marshal error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}
func (c *usersController) balance(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(int)
	balance, err := c.repo.Balances.Get(r.Context(), userID)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(balance)
	if err != nil {
		log.Debug().Err(err).Msg("marshal error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func (c *usersController) withdraw(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(int)
	var wr models.WithdrawRequest
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&wr)
	if err != nil {
		if errors.Is(err, service.ErrFormatOrderNum) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Debug().Err(err).Msgf("can't parse json: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	wr.UID = &userID

	err = c.repo.Balances.Withdraw(r.Context(), wr,)

	if err != nil {
		if errors.Is(err, repository.ErrBalanceNotEnough) {
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (c *usersController) withdrawals(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(int)

	wrRequests, err := c.repo.WithdrawRequests.List(r.Context(), userID)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(wrRequests) < 1 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	body, err := json.Marshal(wrRequests)

	if err != nil {
		log.Debug().Err(err).Msg("marshal error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}
