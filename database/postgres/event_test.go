//go:build integration
// +build integration

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/frain-dev/convoy/database"
	"github.com/frain-dev/convoy/datastore"
	"github.com/frain-dev/convoy/pkg/httpheader"
	"github.com/frain-dev/convoy/util"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func Test_CreateEvent(t *testing.T) {
	db, closeFn := getDB(t)
	defer closeFn()

	eventRepo := NewEventRepo(db)
	event := generateEvent(t, db)
	ctx := context.Background()

	require.NoError(t, eventRepo.CreateEvent(ctx, event))

	newEvent, err := eventRepo.FindEventByID(ctx, event.UID)
	require.NoError(t, err)

	newEvent.CreatedAt = time.Time{}
	newEvent.UpdatedAt = time.Time{}
	event.CreatedAt, event.UpdatedAt = time.Time{}, time.Time{}

	require.Equal(t, event, newEvent)
}

func Test_FindEventByID(t *testing.T) {
	db, closeFn := getDB(t)
	defer closeFn()

	eventRepo := NewEventRepo(db)
	event := generateEvent(t, db)
	ctx := context.Background()

	_, err := eventRepo.FindEventByID(ctx, event.UID)
	require.Error(t, err)
	require.True(t, errors.Is(err, datastore.ErrEventNotFound))

	require.NoError(t, eventRepo.CreateEvent(ctx, event))

	newEvent, err := eventRepo.FindEventByID(ctx, event.UID)
	require.NoError(t, err)

	newEvent.CreatedAt = time.Time{}
	newEvent.UpdatedAt = time.Time{}

	event.CreatedAt, event.UpdatedAt = time.Time{}, time.Time{}
	require.Equal(t, event, newEvent)
}

func Test_FindEventsByIDs(t *testing.T) {
	db, closeFn := getDB(t)
	defer closeFn()

	eventRepo := NewEventRepo(db)
	ctx := context.Background()
	var events []*datastore.Event
	var ids []string

	events = append(events, generateEvent(t, db), generateEvent(t, db))

	for _, event := range events {
		err := eventRepo.CreateEvent(ctx, event)
		require.NoError(t, err)

		ids = append(ids, event.UID)

	}

	records, err := eventRepo.FindEventsByIDs(ctx, ids)
	require.NoError(t, err)

	require.Equal(t, 2, len(records))
}

func Test_CountProjectMessages(t *testing.T) {
	data := json.RawMessage([]byte(`{
		"event_id": "123456",
		"endpoint_id": "123456"
	}`))

	tests := []struct {
		name  string
		count int
		data  json.RawMessage
	}{
		{
			name:  "Count Project Messages - 10 records",
			count: 10,
			data:  data,
		},

		{
			name:  "Count Project Messages - 12 records",
			count: 12,
			data:  data,
		},

		{
			name:  "Count Project Messages - 5 records",
			count: 5,
			data:  data,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, closeFn := getDB(t)
			defer closeFn()

			project := seedProject(t, db)
			endpoint := generateEndpoint(project)
			eventRepo := NewEventRepo(db)

			err := NewEndpointRepo(db).CreateEndpoint(context.Background(), endpoint, project.UID)
			require.NoError(t, err)

			for i := 0; i < tc.count; i++ {
				event := &datastore.Event{
					UID:       ulid.Make().String(),
					EventType: "test-event",
					Endpoints: []string{endpoint.UID},
					ProjectID: project.UID,
					Headers:   httpheader.HTTPHeader{},
					Raw:       string(tc.data),
					Data:      data,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				err := eventRepo.CreateEvent(context.Background(), event)
				require.NoError(t, err)
			}

			count, err := eventRepo.CountProjectMessages(context.Background(), project.UID)

			require.NoError(t, err)
			require.Equal(t, tc.count, int(count))
		})
	}
}

func Test_CountEvents(t *testing.T) {
	data := json.RawMessage([]byte(`{
		"event_id": "123456",
		"endpoint_id": "123456"
	}`))

	tests := []struct {
		name  string
		count int
		data  json.RawMessage
	}{
		{
			name:  "Count Events - 11 records",
			count: 11,
			data:  data,
		},

		{
			name:  "Count Events - 12 records",
			count: 12,
			data:  data,
		},

		{
			name:  "Count Events - 10 records",
			count: 10,
			data:  data,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, closeFn := getDB(t)
			defer closeFn()

			project := seedProject(t, db)
			endpoint := generateEndpoint(project)
			eventRepo := NewEventRepo(db)

			err := NewEndpointRepo(db).CreateEndpoint(context.Background(), endpoint, project.UID)
			require.NoError(t, err)

			for i := 0; i < tc.count; i++ {
				event := &datastore.Event{
					UID:       ulid.Make().String(),
					EventType: "test-event",
					Endpoints: []string{endpoint.UID},
					ProjectID: project.UID,
					Headers:   httpheader.HTTPHeader{},
					Raw:       string(tc.data),
					Data:      data,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				err := eventRepo.CreateEvent(context.Background(), event)
				require.NoError(t, err)
			}

			count, err := eventRepo.CountEvents(context.Background(), &datastore.Filter{
				Project:    project,
				EndpointID: endpoint.UID,
				SearchParams: datastore.SearchParams{
					CreatedAtStart: time.Now().Unix(),
					CreatedAtEnd:   time.Now().Add(5 * time.Minute).Unix(),
				},
			})

			require.NoError(t, err)
			require.Equal(t, tc.count, int(count))
		})
	}
}

func Test_LoadEventsPaged(t *testing.T) {
	type Expected struct {
		paginationData datastore.PaginationData
	}

	data := json.RawMessage([]byte(`{
		"event_id": "123456",
		"endpoint_id": "123456"
	}`))

	tests := []struct {
		name       string
		pageData   datastore.Pageable
		count      int
		endpointID string
		expected   Expected
	}{
		{
			name:     "Load Events Paged - 10 records",
			pageData: datastore.Pageable{Page: 1, PerPage: 3},
			count:    10,
			expected: Expected{
				paginationData: datastore.PaginationData{
					Total:     10,
					TotalPage: 4,
					Page:      1,
					PerPage:   3,
					Prev:      1,
					Next:      2,
				},
			},
		},

		{
			name:     "Load Events Paged - 12 records",
			pageData: datastore.Pageable{Page: 2, PerPage: 4},
			count:    12,
			expected: Expected{
				paginationData: datastore.PaginationData{
					Total:     12,
					TotalPage: 3,
					Page:      2,
					PerPage:   4,
					Prev:      1,
					Next:      3,
				},
			},
		},

		{
			name:     "Load Events Paged - 5 records",
			pageData: datastore.Pageable{Page: 1, PerPage: 3},
			count:    5,
			expected: Expected{
				paginationData: datastore.PaginationData{
					Total:     5,
					TotalPage: 2,
					Page:      1,
					PerPage:   3,
					Prev:      1,
					Next:      2,
				},
			},
		},

		{
			name:       "Filter Events Paged By Endpoint ID - 1 record",
			pageData:   datastore.Pageable{Page: 1, PerPage: 3},
			count:      1,
			endpointID: ulid.Make().String(),
			expected: Expected{
				paginationData: datastore.PaginationData{
					Total:     1,
					TotalPage: 1,
					Page:      1,
					PerPage:   3,
					Prev:      1,
					Next:      2,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, closeFn := getDB(t)
			defer closeFn()

			project := seedProject(t, db)
			endpoint := generateEndpoint(project)
			eventRepo := NewEventRepo(db)

			if !util.IsStringEmpty(tc.endpointID) {
				endpoint.UID = tc.endpointID
			}

			err := NewEndpointRepo(db).CreateEndpoint(context.Background(), endpoint, project.UID)
			require.NoError(t, err)

			for i := 0; i < tc.count; i++ {
				event := &datastore.Event{
					UID:       ulid.Make().String(),
					EventType: "test-event",
					Endpoints: []string{endpoint.UID},
					ProjectID: project.UID,
					Headers:   httpheader.HTTPHeader{},
					Raw:       string(data),
					Data:      data,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				err := eventRepo.CreateEvent(context.Background(), event)
				require.NoError(t, err)
			}

			_, pageable, err := eventRepo.LoadEventsPaged(context.Background(), &datastore.Filter{
				Project:    project,
				EndpointID: endpoint.UID,
				SearchParams: datastore.SearchParams{
					CreatedAtStart: time.Now().Add(-time.Hour).Unix(),
					CreatedAtEnd:   time.Now().Add(5 * time.Minute).Unix(),
				},
				Pageable: tc.pageData,
			})

			require.NoError(t, err)

			require.Equal(t, tc.expected.paginationData.Total, pageable.Total)
			require.Equal(t, tc.expected.paginationData.TotalPage, pageable.TotalPage)
			require.Equal(t, tc.expected.paginationData.Page, pageable.Page)
			require.Equal(t, tc.expected.paginationData.PerPage, pageable.PerPage)
			require.Equal(t, tc.expected.paginationData.Prev, pageable.Prev)
			require.Equal(t, tc.expected.paginationData.Next, pageable.Next)
		})
	}
}

func Test_SoftDeleteProjectEvents(t *testing.T) {
	db, closeFn := getDB(t)
	defer closeFn()

	eventRepo := NewEventRepo(db)
	event := generateEvent(t, db)
	ctx := context.Background()

	require.NoError(t, eventRepo.CreateEvent(ctx, event))

	_, err := eventRepo.FindEventByID(ctx, event.UID)
	require.NoError(t, err)

	require.NoError(t, eventRepo.DeleteProjectEvents(ctx, &datastore.EventFilter{
		ProjectID:      event.ProjectID,
		CreatedAtStart: time.Now().Unix(),
		CreatedAtEnd:   time.Now().Add(5 * time.Minute).Unix(),
	}, false))

	_, err = eventRepo.FindEventByID(ctx, event.UID)
	require.Error(t, err)
	require.True(t, errors.Is(err, datastore.ErrEventNotFound))
}

func Test_HardDeleteProjectEvents(t *testing.T) {
	db, closeFn := getDB(t)
	defer closeFn()

	eventRepo := NewEventRepo(db)
	event := generateEvent(t, db)
	ctx := context.Background()

	require.NoError(t, eventRepo.CreateEvent(ctx, event))

	_, err := eventRepo.FindEventByID(ctx, event.UID)
	require.NoError(t, err)

	require.NoError(t, eventRepo.DeleteProjectEvents(ctx, &datastore.EventFilter{
		ProjectID:      event.ProjectID,
		CreatedAtStart: time.Now().Unix(),
		CreatedAtEnd:   time.Now().Add(5 * time.Minute).Unix(),
	}, true))

	_, err = eventRepo.FindEventByID(ctx, event.UID)
	require.Error(t, err)
	require.True(t, errors.Is(err, datastore.ErrEventNotFound))
}

func generateEvent(t *testing.T, db database.Database) *datastore.Event {
	project := seedProject(t, db)
	endpoint := generateEndpoint(project)

	err := NewEndpointRepo(db).CreateEndpoint(context.Background(), endpoint, project.UID)
	require.NoError(t, err)

	data := json.RawMessage([]byte(`{
		"event_id": "123456",
		"endpoint_id": "123456"
	}`))

	return &datastore.Event{
		UID:       ulid.Make().String(),
		EventType: "test-event",
		Endpoints: []string{endpoint.UID},
		ProjectID: project.UID,
		Headers:   httpheader.HTTPHeader{},
		Raw:       string(data),
		Data:      data,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func seedEvent(t *testing.T, db database.Database, project *datastore.Project) *datastore.Event {
	ev := generateEvent(t, db)
	ev.ProjectID = project.UID

	require.NoError(t, NewEventRepo(db).CreateEvent(context.Background(), ev))
	return ev
}
