package handler

import(
	"net/http"
	"net/http/httptest"
	"testing"
	"strings"
	"io"
	"github.com/stretchr/testify/assert"
	"github.com/go-chi/chi/v5"
	gomock "go.uber.org/mock/gomock"
	"context"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/mock_repository"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/repository"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/models"
	"os/signal"
	"syscall"
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"time"
	"encoding/json"
)

type want struct {
	code int
	contentType string
	authHeader string
	body string
}

type bodyCase struct {
	name string
	body string
	want want
}

type pathCase struct {
	name string
	path string
	bodyCases []bodyCase
}

type methodCase struct {
	name string
	method string
	pathCases []pathCase
}

type requestCase struct {
	name string
	contentType string
	authorization string
	methodCases []methodCase
	token *jwt.Token
}

type serverCase struct {
	name string
	sk any
	requestCases []requestCase
}

func TestNewUsersController(t *testing.T) {
	t.Run("create new users controller", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockR := repository.Repository{
			Users: mock_repository.NewMockUserRepository(ctrl),
			Orders: mock_repository.NewMockOrderRepository(ctrl),
			Balances: mock_repository.NewMockBalanceRepository(ctrl),
			WithdrawRequests: mock_repository.NewMockWithdrawRequestRepository(ctrl),
		}
		uc := NewUsersController(&mockR, nil)

		assert.NotNil(t, uc)
	})
}

func TestUsersRegister(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockUR := mock_repository.NewMockUserRepository(ctrl)
	mockR := repository.Repository{
		Users: mockUR,
	}
	mockUR.
		EXPECT().
		Register(gomock.Any(), gomock.Eq(models.User{Login: "user1", Password: "11111111"})).
		MaxTimes(4).
		Return(1, nil)

	mockUR.
		EXPECT().
		Register(gomock.Any(), gomock.Eq(models.User{Login: "user2", Password: "11111111"})).
		MaxTimes(2).
		Return(0, repository.ErrLoginExist)
	
	mockUR.
		EXPECT().
		Register(gomock.Any(), gomock.Eq(models.User{Login: "user3", Password: "11111111"})).
		MaxTimes(2).
		Return(0, errors.New("any ERROR"))

	testCases := []serverCase{
		serverCase{
			name: "without secret key",
			sk: nil,
			requestCases: []requestCase {
				requestCase{
					name: "Content-Type: application/json;",
					contentType: "application/json",
					methodCases: []methodCase{
						methodCase{
							name: "Method: POST",
							method: http.MethodPost,
							pathCases: []pathCase{
								pathCase{
									name: "user register path",
									path: "/api/user/register",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with empty body",
											want: want{ code: http.StatusBadRequest, contentType: "", body: ""},
										},
										bodyCase{
											name: "with bad json",
											body: "{login: ",
											want: want{ code: http.StatusBadRequest, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid json",
											body: "{\"login\": \"user1\", \"password\": \"11111111\"}",
											want: want{ code: http.StatusInternalServerError, contentType: "",},
										},
										bodyCase{
											name: "with valid json, but user exist",
											body: "{\"login\": \"user2\", \"password\": \"11111111\"}",
											want: want{ code: http.StatusConflict, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid json, but something went wrong with repo register",
											body: "{\"login\": \"user3\", \"password\": \"11111111\"}",
											want: want{ code: http.StatusInternalServerError, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
				
			},
		},
		serverCase{
			name: "with secret key",
			sk: []byte(nil),
			requestCases: []requestCase{
				requestCase{
					name: "Content-Type: application/json",
					contentType: "application/json",
					methodCases: []methodCase{
						methodCase{
							name: "Method: POST",
							method: http.MethodPost,
							pathCases: []pathCase{
								pathCase{
									name: "user register path",
									path: "/api/user/register",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with empty body",
											want: want{ code: http.StatusBadRequest, contentType: "", body: ""},
										},
										bodyCase{
											name: "with bad json",
											body: "{login: ",
											want: want{ code: http.StatusBadRequest, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid json",
											body: "{\"login\": \"user1\", \"password\": \"11111111\"}",
											want: want{ code: http.StatusOK, contentType: "", body: "Success", authHeader: "Bearer "},
										},
										bodyCase{
											name: "with valid json, but user exist",
											body: "{\"login\": \"user2\", \"password\": \"11111111\"}",
											want: want{ code: http.StatusConflict, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid json, but something went wrong with repo register",
											body: "{\"login\": \"user3\", \"password\": \"11111111\"}",
											want: want{ code: http.StatusInternalServerError, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	runCases(ctx, t, testCases, &mockR)
}

func TestUsersLogin(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockUR := mock_repository.NewMockUserRepository(ctrl)
	mockR := repository.Repository{
		Users: mockUR,
	}
	mockUR.
		EXPECT().
		Auth(gomock.Any(), gomock.Eq(models.User{Login: "user1", Password: "11111111"})).
		MaxTimes(2).
		Return(1, nil)

	mockUR.
		EXPECT().
		Auth(gomock.Any(), gomock.Eq(models.User{Login: "user2", Password: "1111111"})).
		MaxTimes(2).
		Return(0, repository.ErrWrongCredentials)
	
	mockUR.
		EXPECT().
		Auth(gomock.Any(), gomock.Eq(models.User{Login: "user3", Password: "11111111"})).
		MaxTimes(2).
		Return(0, errors.New("any ERROR"))

	testCases := []serverCase {
		serverCase{
			name: "without secret key",
			sk: nil,
			requestCases: []requestCase{
				requestCase{
					name: "Content-Type: application/json; server without secret key",
					contentType: "application/json",
					methodCases: []methodCase{
						methodCase{
							name: "Method: POST",
							method: http.MethodPost,
							pathCases: []pathCase{
								pathCase{
									name: "user login path",
									path: "/api/user/login",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with empty body",
											want: want{ code: http.StatusBadRequest, contentType: "", body: ""},
										},
										bodyCase{
											name: "with bad json",
											body: "{login: ",
											want: want{ code: http.StatusBadRequest, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid json",
											body: "{\"login\": \"user1\", \"password\": \"11111111\"}",
											want: want{ code: http.StatusInternalServerError, contentType: "",},
										},
										bodyCase{
											name: "with valid json, but bad credentials",
											body: "{\"login\": \"user2\", \"password\": \"1111111\"}",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid json, but something went wrong with repo register",
											body: "{\"login\": \"user3\", \"password\": \"11111111\"}",
											want: want{ code: http.StatusInternalServerError, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		serverCase {
			name: "with secret key",
			sk: []byte(nil),
			requestCases: []requestCase{
				requestCase{
					name: "Content-Type: application/json",
					contentType: "application/json",
					methodCases: []methodCase{
						methodCase{
							name: "Method: POST",
							method: http.MethodPost,
							pathCases: []pathCase{
								pathCase{
									name: "user login path",
									path: "/api/user/login",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with empty body",
											want: want{ code: http.StatusBadRequest, contentType: "", body: ""},
										},
										bodyCase{
											name: "with bad json",
											body: "{login: ",
											want: want{ code: http.StatusBadRequest, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid json",
											body: "{\"login\": \"user1\", \"password\": \"11111111\"}",
											want: want{ code: http.StatusOK, contentType: "", body: "Success", authHeader: "Bearer "},
										},
										bodyCase{
											name: "with valid json, but bad credentials",
											body: "{\"login\": \"user2\", \"password\": \"1111111\"}",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid json, but something went wrong with repo register",
											body: "{\"login\": \"user3\", \"password\": \"11111111\"}",
											want: want{ code: http.StatusInternalServerError, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	runCases(ctx, t, testCases, &mockR)
}

func TestPostOrders(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUR := mock_repository.NewMockUserRepository(ctrl)
	mockOR := mock_repository.NewMockOrderRepository(ctrl)
	mockR := repository.Repository{
		Users: mockUR,
		Orders: mockOR,
	}
	mUID := 1
	mNum := "12345674"
	loadedMNum := "7992738"
	loadedByOtherNum := "49927398716"

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims {
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		UserID: mUID,
	})

	jwtTokenExpired := jwt.NewWithClaims(jwt.SigningMethodHS256, claims {
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP * -1)),
		},
		UserID: mUID,
	})

	mockOR.
		EXPECT().
		Create(gomock.Any(), gomock.Eq(models.Order{UID: &mUID, Number: mNum})).
		Return(nil)

	mockOR.
		EXPECT().
		Create(gomock.Any(), gomock.Eq(models.Order{UID: &mUID, Number: loadedMNum})).
		Return(repository.ErrOrderExist)

	mockOR.
		EXPECT().
		Create(gomock.Any(), gomock.Eq(models.Order{UID: &mUID, Number: loadedByOtherNum})).
		Return(repository.ErrOrderLoadedByOtherUser)

	mockUR.
		EXPECT().
		ExistByID(gomock.Any(), gomock.Eq(1)).
		AnyTimes().
		Return(true, nil)

	testCases := []serverCase {
		serverCase{
			name: "without secret key",
			sk: nil,
			requestCases: []requestCase{
				requestCase{
					name: "Content-Type: text/plain, Authorization empty",
					contentType: "text/plain",
					methodCases: []methodCase{
						methodCase{
							name: "Method: POST",
							method: http.MethodPost,
							pathCases: []pathCase{
								pathCase{
									name: "user load order path",
									path: "/api/user/orders",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with empty body",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid body",
											body: "12345674",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid body, but order already loaded",
											body: "7992739",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
										bodyCase{
											name: "with bad body",
											body: "1234s213",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid body, but bad format order num",
											body: "12345677",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		serverCase{
			name: "with secret key",
			sk: []byte(nil),
			requestCases: []requestCase{
				requestCase{
					name: "Content-Type: text/plain, Authorization empty",
					contentType: "text/plain",
					methodCases: []methodCase{
						methodCase{
							name: "Method: POST",
							method: http.MethodPost,
							pathCases: []pathCase{
								pathCase{
									name: "user load order path",
									path: "/api/user/orders",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with empty body",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid body",
											body: "12345674",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid body, but order already loaded",
											body: "7992739",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
										bodyCase{
											name: "with bad body",
											body: "1234s213",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid body, but bad format order num",
											body: "12345677",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
				requestCase{
					name: "Content-Type: text/plain, Authentication exist, but token expired",
					contentType: "text/plain",
					authorization: "Bearer ",
					token: jwtTokenExpired,
					methodCases: []methodCase{
						methodCase{
							name: "Method: POST",
							method: http.MethodPost,
							pathCases: []pathCase{
								pathCase{
									name: "user load order path",
									path: "/api/user/orders",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with empty body and empty ",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid body",
											body: "12345674",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid body, but order already loaded",
											body: "7992738",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
										bodyCase{
											name: "with bad body",
											body: "1234s213",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid body, but bad format order num",
											body: "12345677",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
				requestCase{
					name: "Content-Type: text/plain",
					contentType: "text/plain",
					authorization: "Bearer ",
					token: jwtToken,
					methodCases: []methodCase{
						methodCase{
							name: "Method: POST",
							method: http.MethodPost,
							pathCases: []pathCase{
								pathCase{
									name: "user load order path",
									path: "/api/user/orders",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with empty body",
											want: want{ code: http.StatusBadRequest, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid body",
											body: "12345674",
											want: want{ code: http.StatusAccepted, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid body, but order already loaded",
											body: "7992738",
											want: want{ code: http.StatusOK, contentType: "", body: ""},
										},
										bodyCase{
											name: "with bad body",
											body: "1234s213",
											want: want{ code: http.StatusBadRequest, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid body, but bad format order num",
											body: "12345677",
											want: want{ code: http.StatusUnprocessableEntity, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid body, but order was loaded by other user",
											body: "49927398716",
											want: want{ code: http.StatusConflict, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	
	runCases(ctx, t, testCases, &mockR)
}

func TestGetOrders(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUR := mock_repository.NewMockUserRepository(ctrl)
	mockOR := mock_repository.NewMockOrderRepository(ctrl)

	mockR := repository.Repository{
		Users: mockUR,
		Orders: mockOR,
	}
	mUID := 1
	mUID2 := 2
	mNum1 := "12345674"
	mNum2 := "7992738"
	mOrders := []models.Order{
		models.Order{UID: &mUID2, Number: mNum1, Status: "NEW", Accrual: new(float64)},
		models.Order{UID: &mUID2, Number: mNum2, Status: "NEW", Accrual: new(float64)},
}

	marshalOrders, err := json.Marshal(mOrders)
	if err != nil { t.Fatal(err) }

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims {
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		UserID: mUID,
	})

	jwtToken2 := jwt.NewWithClaims(jwt.SigningMethodHS256, claims {
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		UserID: mUID2,
	})

	jwtTokenExpired := jwt.NewWithClaims(jwt.SigningMethodHS256, claims {
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP * -1)),
		},
		UserID: mUID,
	})

	mockOR.
		EXPECT().
		List(gomock.Any(), gomock.Eq(mUID)).
		Return(nil, nil)

	mockOR.
		EXPECT().
		List(gomock.Any(), gomock.Eq(mUID2)).
		Return(mOrders, nil)

	mockUR.
		EXPECT().
		ExistByID(gomock.Any(), gomock.Eq(mUID)).
		AnyTimes().
		Return(true, nil)

	mockUR.
		EXPECT().
		ExistByID(gomock.Any(), gomock.Eq(mUID2)).
		Return(true, nil)

	testCases := []serverCase {
		serverCase{
			name: "without secret key",
			sk: nil,
			requestCases: []requestCase{
				requestCase{
					name: "Content-Type: text/plain, Authorization empty",
					contentType: "text/plain",
					methodCases: []methodCase{
						methodCase{
							name: "Method: GET",
							method: http.MethodGet,
							pathCases: []pathCase{
								pathCase{
									name: "user list orders path",
									path: "/api/user/orders",
									bodyCases: []bodyCase{
										bodyCase{
											name: "any body",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		serverCase{
			name: "with secret key",
			sk: []byte(nil),
			requestCases: []requestCase{
				requestCase{
					name: "Content-Type: text/plain, Authorization empty",
					contentType: "text/plain",
					methodCases: []methodCase{
						methodCase{
							name: "Method: GET",
							method: http.MethodGet,
							pathCases: []pathCase{
								pathCase{
									name: "user list orders path",
									path: "/api/user/orders",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with any body",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
				requestCase{
					name: "Content-Type: text/plain, Authentication exist, but token expired",
					contentType: "text/plain",
					authorization: "Bearer ",
					token: jwtTokenExpired,
					methodCases: []methodCase{
						methodCase{
							name: "Method: GET",
							method: http.MethodGet,
							pathCases: []pathCase{
								pathCase{
									name: "user list orders path",
									path: "/api/user/orders",
									bodyCases: []bodyCase{
										bodyCase{
											name: "any body",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
				requestCase{
					name: "Content-Type: text/plain, but orders empty",
					contentType: "text/plain",
					authorization: "Bearer ",
					token: jwtToken,
					methodCases: []methodCase{
						methodCase{
							name: "Method: GET",
							method: http.MethodGet,
							pathCases: []pathCase{
								pathCase{
									name: "user list orders path",
									path: "/api/user/orders",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with any body",
											want: want{ code: http.StatusNoContent, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
				requestCase{
					name: "Content-Type: text/plain",
					contentType: "text/plain",
					authorization: "Bearer ",
					token: jwtToken2,
					methodCases: []methodCase{
						methodCase{
							name: "Method: GET",
							method: http.MethodGet,
							pathCases: []pathCase{
								pathCase{
									name: "user list orders path",
									path: "/api/user/orders",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with any body",
											want: want{ code: http.StatusOK, contentType: "application/json", body: string(marshalOrders)},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	
	runCases(ctx, t, testCases, &mockR)
}

func TestBalance(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUR := mock_repository.NewMockUserRepository(ctrl)
	mockBR := mock_repository.NewMockBalanceRepository(ctrl)

	mockR := repository.Repository{
		Users: mockUR,
		Balances: mockBR,
	}
	mUID := 1

	mBalance := models.Balance{Current: new(float64), Withdrawn: new(float64)}

	marshalBalance, err := json.Marshal(mBalance)

	if err != nil { t.Fatal(err) }

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims {
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		UserID: mUID,
	})

	mockUR.
		EXPECT().
		ExistByID(gomock.Any(), gomock.Eq(mUID)).
		AnyTimes().
		Return(true, nil)

	mockBR.
		EXPECT().
		Get(gomock.Any(), gomock.Eq(mUID)).
		AnyTimes().
		Return(&models.Balance{Current: new(float64), Withdrawn: new(float64)}, nil)

	testCases := []serverCase {
		serverCase{
			name: "without secret key",
			sk: nil,
			requestCases: []requestCase{
				requestCase{
					name: "Content-Type: text/plain, Authorization empty",
					contentType: "text/plain",
					methodCases: []methodCase{
						methodCase{
							name: "Method: GET",
							method: http.MethodGet,
							pathCases: []pathCase{
								pathCase{
									name: "user balance path",
									path: "/api/user/balance",
									bodyCases: []bodyCase{
										bodyCase{
											name: "any body",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		serverCase{
			name: "with secret key",
			sk: []byte(nil),
			requestCases: []requestCase{
				requestCase{
					name: "Content-Type: text/plain, Authorization empty",
					contentType: "text/plain",
					methodCases: []methodCase{
						methodCase{
							name: "Method: GET",
							method: http.MethodGet,
							pathCases: []pathCase{
								pathCase{
									name: "user balance path",
									path: "/api/user/balance",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with any body",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
				requestCase{
					name: "Content-Type: text/plain",
					contentType: "text/plain",
					authorization: "Bearer ",
					token: jwtToken,
					methodCases: []methodCase{
						methodCase{
							name: "Method: GET",
							method: http.MethodGet,
							pathCases: []pathCase{
								pathCase{
									name: "user balance path",
									path: "/api/user/balance",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with any body",
											want: want{ code: http.StatusOK, contentType: "application/json", body: string(marshalBalance)},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	
	runCases(ctx, t, testCases, &mockR)
}

func TestWithdraw(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUR := mock_repository.NewMockUserRepository(ctrl)
	mockBR := mock_repository.NewMockBalanceRepository(ctrl)
	mockWRR := mock_repository.NewMockWithdrawRequestRepository(ctrl)

	mockR := repository.Repository{
		Users: mockUR,
		Balances: mockBR,
		WithdrawRequests: mockWRR,
	}
	mUID := 1

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims {
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		UserID: mUID,
	})

	mockUR.
		EXPECT().
		ExistByID(gomock.Any(), gomock.Eq(mUID)).
		AnyTimes().
		Return(true, nil)

	orderNum := "12345674"
	withdrawSum := 767.7
	wrR := models.WithdrawRequest{UID: &mUID, Order: orderNum, Sum: &withdrawSum}

	mockBR.
		EXPECT().
		Withdraw(gomock.Any(), gomock.Eq(wrR)).
		Return(nil).
		Do(func(ctx context.Context, wr models.WithdrawRequest) {
			mockWRR.Create(ctx, wr)
		})

	withdrawSum2 := 1000.0

	mockBR.
		EXPECT().
		Withdraw(gomock.Any(), gomock.Eq(models.WithdrawRequest{UID: &mUID, Order: orderNum, Sum: &withdrawSum2})).
		Return(repository.ErrBalanceNotEnough)

	mockWRR.
		EXPECT().
		Create(gomock.Any(), gomock.Eq(wrR)).
		Return(nil)

	testCases := []serverCase {
		serverCase{
			name: "without secret key",
			sk: nil,
			requestCases: []requestCase{
				requestCase{
					name: "Content-Type: application/json, Authorization empty",
					contentType: "application/json",
					methodCases: []methodCase{
						methodCase{
							name: "Method: POST",
							method: http.MethodPost,
							pathCases: []pathCase{
								pathCase{
									name: "user balance withdraw path",
									path: "/api/user/balance/withdraw",
									bodyCases: []bodyCase{
										bodyCase{
											name: "any body",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		serverCase{
			name: "with secret key",
			sk: []byte(nil),
			requestCases: []requestCase{
				requestCase{
					name: "Content-Type: application/json, Authorization empty",
					contentType: "application/json",
					methodCases: []methodCase{
						methodCase{
							name: "Method: POST",
							method: http.MethodPost,
							pathCases: []pathCase{
								pathCase{
									name: "user balance withdraw path",
									path: "/api/user/balance/withdraw",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with any body",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
				requestCase{
					name: "Content-Type: application/json",
					contentType: "application/json",
					authorization: "Bearer ",
					token: jwtToken,
					methodCases: []methodCase{
						methodCase{
							name: "Method: POST",
							method: http.MethodPost,
							pathCases: []pathCase{
								pathCase{
									name: "user balance withdraw path",
									path: "/api/user/balance/withdraw",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with valid body",
											body: "{\"order\": \"12345674\", \"sum\": 767.7}",
											want: want{ code: http.StatusOK, contentType: "", body: ""},
										},
										bodyCase{
											name: "with invalid body",
											body: "{\"order\": \"12345674\" \"sum\": 767.7}",
											want: want{ code: http.StatusBadRequest, contentType: "", body: ""},
										},
										bodyCase{
											name: "with valid body, but not enough balance",
											body: "{\"order\": \"12345674\", \"sum\": 1000}",
											want: want{ code: http.StatusPaymentRequired, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	
	runCases(ctx, t, testCases, &mockR)
}

func TestWithdrawls(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUR := mock_repository.NewMockUserRepository(ctrl)
	mockWRR := mock_repository.NewMockWithdrawRequestRepository(ctrl)

	mockR := repository.Repository{
		Users: mockUR,
		WithdrawRequests: mockWRR,
	}
	mUID := 1
	mUID2 := 2

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims {
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		UserID: mUID,
	})

	jwtToken2 := jwt.NewWithClaims(jwt.SigningMethodHS256, claims {
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		UserID: mUID2,
	})

	mockUR.
		EXPECT().
		ExistByID(gomock.Any(), gomock.Eq(mUID)).
		AnyTimes().
		Return(true, nil)

	mockUR.
		EXPECT().
		ExistByID(gomock.Any(), gomock.Eq(mUID2)).
		AnyTimes().
		Return(true, nil)


	wrOrderNum := "12345674"
	wrOrderNum2 := "7992738"
	wrSum := 500.0
	wrSum2 := 100.0

	mWRs := []models.WithdrawRequest{
		models.WithdrawRequest{Order: wrOrderNum, Sum: &wrSum},
		models.WithdrawRequest{Order: wrOrderNum2, Sum: &wrSum2},
	}

	marshaledWRs, err := json.Marshal(mWRs)

	if err != nil { t.Fatal(err) }

	mockWRR.
		EXPECT().
		List(gomock.Any(), gomock.Eq(mUID)).
		Return(mWRs, nil)
	mockWRR.
		EXPECT().
		List(gomock.Any(), gomock.Eq(mUID2)).
		Return(nil, nil)

	testCases := []serverCase {
		serverCase{
			name: "without secret key",
			sk: nil,
			requestCases: []requestCase{
				requestCase{
					name: "Content-Type: application/json, Authorization empty",
					contentType: "application/json",
					methodCases: []methodCase{
						methodCase{
							name: "Method: GET",
							method: http.MethodGet,
							pathCases: []pathCase{
								pathCase{
									name: "user withdrawals path",
									path: "/api/user/withdrawals",
									bodyCases: []bodyCase{
										bodyCase{
											name: "any body",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		serverCase{
			name: "with secret key",
			sk: []byte(nil),
			requestCases: []requestCase{
				requestCase{
					name: "Content-Type: application/json, Authorization empty",
					contentType: "application/json",
					methodCases: []methodCase{
						methodCase{
							name: "Method: GET",
							method: http.MethodGet,
							pathCases: []pathCase{
								pathCase{
									name: "user withdrawals path",
									path: "/api/user/withdrawals",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with any body",
											want: want{ code: http.StatusUnauthorized, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
				requestCase{
					name: "Content-Type: application/json",
					contentType: "application/json",
					authorization: "Bearer ",
					token: jwtToken,
					methodCases: []methodCase{
						methodCase{
							name: "Method: GET",
							method: http.MethodGet,
							pathCases: []pathCase{
								pathCase{
									name: "user withdrawals path",
									path: "/api/user/withdrawals",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with valid body",
											want: want{ code: http.StatusOK, contentType: "", body: string(marshaledWRs)},
										},
									},
								},
							},
						},
					},
				},
				requestCase{
					name: "Content-Type: application/json",
					contentType: "application/json",
					authorization: "Bearer ",
					token: jwtToken2,
					methodCases: []methodCase{
						methodCase{
							name: "Method: GET",
							method: http.MethodGet,
							pathCases: []pathCase{
								pathCase{
									name: "user withdrawals path",
									path: "/api/user/withdrawals",
									bodyCases: []bodyCase{
										bodyCase{
											name: "with valid body",
											want: want{ code: http.StatusNoContent, contentType: "", body: ""},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	
	runCases(ctx, t, testCases, &mockR)
}

func runCases(ctx context.Context, t *testing.T, cases []serverCase, repo *repository.Repository) {
	for _, tc := range cases {
		for _, rtc := range tc.requestCases {
			if rtc.token != nil {
				signedStr, err := rtc.token.SignedString(tc.sk)
				if err != nil { t.Fatal(err) }
				rtc.authorization += signedStr
			}
			for _, mtc := range rtc.methodCases {
				for _, ptc := range mtc.pathCases {
					for _, btc := range ptc.bodyCases {
						t.Run(strings.Join([]string{tc.name, rtc.name, mtc.name, ptc.name, btc.name}, "; "), func(t *testing.T) {
							mux := chi.NewRouter()
							c := NewUsersController(repo, tc.sk)
							c.ApplyTo(mux)
							r := httptest.NewRequest(mtc.method, ptc.path, strings.NewReader(btc.body))
							r.Header.Set("Content-Type", rtc.contentType)
							r.Header.Set("Authorization", rtc.authorization)
							w := httptest.NewRecorder()
							mux.ServeHTTP(w, r)
							rBody, err := io.ReadAll(w.Body)
							if err != nil {
								t.Fatal(err)
							}
							rContentType := w.Header().Get("Content-Type")
							rAuthorization := w.Header().Get("Authorization")
							assert.Equal(t, btc.want.code, w.Code, "Код ответа не совпадает с ожидаемым")
							assert.Equal(t, btc.want.body, string(rBody), "Тело ответа не совпадает с ожидаемым")
							assert.Equal(t, true, strings.HasPrefix(rContentType, btc.want.contentType), "Content-Type ответа не совпадает с ожидаемым")
							assert.Equal(t, true, strings.HasPrefix(rAuthorization, btc.want.authHeader), "Authorization ответа не совпадает с ожидаемым")
						})
					}
				}
			}
		}
	}
}
