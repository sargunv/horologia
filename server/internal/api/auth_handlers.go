package api

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"golang.org/x/crypto/bcrypt"

	apigen "github.com/sargunv/tend/server/internal/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

// sentinelHash is a pre-computed bcrypt hash used to prevent timing-based
// email enumeration. When the email is not found or has no password, we still
// run a bcrypt comparison against this hash so the response time is constant.
var sentinelHash, _ = bcrypt.GenerateFromPassword([]byte("sentinel"), bcrypt.DefaultCost)

// --- Auth: Login ---

func (h *Handler) AuthLogin(ctx context.Context, req *apigen.LoginRequest) (*apigen.LoginResponse, error) {
	q := dbgen.New(h.DB)
	user, err := q.GetUserByEmail(ctx, req.Email)
	if errors.Is(err, sql.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(sentinelHash, []byte(req.Password))
		return nil, badRequest("invalid email or password")
	}
	if err != nil {
		return nil, err
	}

	hashToCompare := sentinelHash
	if user.PasswordHash != nil {
		hashToCompare = []byte(*user.PasswordHash)
	}

	if err := bcrypt.CompareHashAndPassword(hashToCompare, []byte(req.Password)); err != nil {
		return nil, badRequest("invalid email or password")
	}

	raw, hash, err := generateToken()
	if err != nil {
		return nil, err
	}

	_, err = q.CreateAuthToken(ctx, dbgen.CreateAuthTokenParams{
		UserID:    user.ID,
		TokenHash: hash,
		Name:      "",
		Kind:      "session",
		CreatedAt: now(),
	})
	if err != nil {
		return nil, err
	}

	apiUser, err := userFromDB(user)
	if err != nil {
		return nil, err
	}

	return &apigen.LoginResponse{
		Token: raw,
		User:  *apiUser,
	}, nil
}

// --- Auth: Tokens ---

func (h *Handler) AuthListTokens(ctx context.Context, params apigen.AuthListTokensParams) (*apigen.AuthTokenPage, error) {
	user := UserFromContext(ctx)

	cursorID, err := decodeCursorInt64(params.Cursor)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	limit := clampLimit(params.Limit)
	q := dbgen.New(h.DB)

	tokens, err := q.ListAuthTokensByUser(ctx, dbgen.ListAuthTokensByUserParams{
		UserID: user.ID,
		ID:     cursorID,
		Limit:  limit + 1,
	})
	if err != nil {
		return nil, err
	}

	items, nextCursor, err := paginate(tokens, limit, authTokenFromDB, func(t dbgen.AuthToken) string {
		return strconv.FormatInt(t.ID, 10)
	})
	if err != nil {
		return nil, err
	}

	return &apigen.AuthTokenPage{Items: items, NextCursor: nextCursor}, nil
}

func (h *Handler) AuthCreateToken(ctx context.Context, req *apigen.AuthTokenCreate) (*apigen.AuthTokenCreateResponse, error) {
	user := UserFromContext(ctx)

	raw, hash, err := generateToken()
	if err != nil {
		return nil, err
	}

	q := dbgen.New(h.DB)
	token, err := q.CreateAuthToken(ctx, dbgen.CreateAuthTokenParams{
		UserID:    user.ID,
		TokenHash: hash,
		Name:      req.Name,
		Kind:      "api",
		CreatedAt: now(),
	})
	if err != nil {
		return nil, err
	}

	apiToken, err := authTokenFromDB(token)
	if err != nil {
		return nil, err
	}

	return &apigen.AuthTokenCreateResponse{
		Token:     raw,
		AuthToken: *apiToken,
	}, nil
}

func (h *Handler) AuthDeleteToken(ctx context.Context, params apigen.AuthDeleteTokenParams) error {
	user := UserFromContext(ctx)

	id, err := parseTokenID(params.TokenId)
	if err != nil {
		return badRequest(err.Error())
	}

	q := dbgen.New(h.DB)
	result, err := q.DeleteAuthToken(ctx, dbgen.DeleteAuthTokenParams{
		ID:     id,
		UserID: user.ID,
	})
	if err != nil {
		return err
	}
	return checkDeleted(result)
}
