package graphql

import (
	"context"
	"net/http"
	"time"

	"github.com/vikstrous/dataloadgen"

	"thesis/app/internal/domain"
)

type ordersKey struct {
	UserID int64
	Limit  int
}

// Loaders: dataloadery tworzone per żądanie (wariant główny — stała liczba zapytań SQL).
type Loaders struct {
	Orders     *dataloadgen.Loader[ordersKey, []domain.Order]
	Items      *dataloadgen.Loader[int64, []domain.OrderItem]
	Categories *dataloadgen.Loader[int64, *domain.Category]
	Users      *dataloadgen.Loader[int64, *domain.User]
}

type ctxKey struct{}

func newLoaders(svc domain.Service) *Loaders {
	// okno łączenia zapytań 1 ms: dodaje do latencji GraphQL do ~1 ms na poziom zagnieżdżenia (opisać w pracy).
	opt := dataloadgen.WithWait(time.Millisecond)
	return &Loaders{
		Orders: dataloadgen.NewLoader(func(ctx context.Context, keys []ordersKey) ([][]domain.Order, []error) {
			out := make([][]domain.Order, len(keys))
			errs := make([]error, len(keys))
			byLimit := map[int][]int64{}
			for _, k := range keys {
				byLimit[k.Limit] = append(byLimit[k.Limit], k.UserID)
			}
			res := map[int]map[int64][]domain.Order{}
			var firstErr error
			for lim, ids := range byLimit {
				m, err := svc.OrdersByUsers(ctx, ids, lim)
				if err != nil {
					firstErr = err
					break
				}
				res[lim] = m
			}
			for i, k := range keys {
				if firstErr != nil {
					errs[i] = firstErr
					continue
				}
				out[i] = res[k.Limit][k.UserID]
			}
			return out, errs
		}, opt),
		Items: dataloadgen.NewLoader(func(ctx context.Context, ids []int64) ([][]domain.OrderItem, []error) {
			m, err := svc.ItemsByOrders(ctx, ids)
			return pick(ids, m, err)
		}, opt),
		Categories: dataloadgen.NewLoader(func(ctx context.Context, ids []int64) ([]*domain.Category, []error) {
			m, err := svc.CategoriesByIDs(ctx, ids)
			return pick(ids, m, err)
		}, opt),
		Users: dataloadgen.NewLoader(func(ctx context.Context, ids []int64) ([]*domain.User, []error) {
			m, err := svc.UsersByIDs(ctx, ids)
			return pick(ids, m, err)
		}, opt),
	}
}

func pick[V any](ids []int64, m map[int64]V, err error) ([]V, []error) {
	out := make([]V, len(ids))
	errs := make([]error, len(ids))
	for i, id := range ids {
		if err != nil {
			errs[i] = err
			continue
		}
		out[i] = m[id]
	}
	return out, errs
}

// LoaderMiddleware wstrzykuje nowe loadery do kontekstu każdego żądania.
func LoaderMiddleware(svc domain.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, newLoaders(svc))))
	})
}

func For(ctx context.Context) *Loaders { return ctx.Value(ctxKey{}).(*Loaders) }
