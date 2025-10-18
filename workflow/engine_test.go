package workflow

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/runlite/db/repository"
	"github.com/dmitrymomot/runlite/testing/mocks"
)

type mockHandler struct {
	mock.Mock
}

func (m *mockHandler) Handle(ctx context.Context, deployment *repository.Deployment) error {
	args := m.Called(ctx, deployment)
	return args.Error(0)
}

func TestNewEngine(t *testing.T) {
	t.Parallel()

	t.Run("with nil logger", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		pollInterval := 5 * time.Second

		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: pollInterval,
		})

		require.NotNil(t, engine)
		require.NotNil(t, engine.logger)
		require.Equal(t, mockRepo, engine.repo)
		require.Equal(t, pollInterval, engine.pollInterval)
		require.NotNil(t, engine.handlers)
		require.NotNil(t, engine.timeouts)
	})

	t.Run("with non-nil logger", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		pollInterval := 10 * time.Second
		logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: pollInterval,
			Logger:       logger,
		})

		require.NotNil(t, engine)
		require.Equal(t, logger, engine.logger)
		require.Equal(t, mockRepo, engine.repo)
		require.Equal(t, pollInterval, engine.pollInterval)
		require.NotNil(t, engine.handlers)
		require.NotNil(t, engine.timeouts)
	})

	t.Run("verify all fields initialized correctly", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		pollInterval := 1 * time.Second
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: pollInterval,
			Logger:       logger,
		})

		require.NotNil(t, engine.repo)
		require.NotNil(t, engine.handlers)
		require.Empty(t, engine.handlers)
		require.NotNil(t, engine.timeouts)
		require.NotNil(t, engine.logger)
		require.Equal(t, pollInterval, engine.pollInterval)
	})
}

func TestEngine_RegisterHandler(t *testing.T) {
	t.Parallel()

	t.Run("successful handler registration", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		handler := &mockHandler{}
		err := engine.RegisterHandler(StatusPending, handler)

		require.NoError(t, err)
		require.Equal(t, handler, engine.handlers[StatusPending])
	})

	t.Run("error when registering duplicate handler for same status", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		handler1 := &mockHandler{}
		handler2 := &mockHandler{}

		err := engine.RegisterHandler(StatusPending, handler1)
		require.NoError(t, err)

		err = engine.RegisterHandler(StatusPending, handler2)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrHandlerAlreadyRegistered)
		require.Equal(t, handler1, engine.handlers[StatusPending])
	})

	t.Run("register handlers for multiple statuses", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		handler1 := &mockHandler{}
		handler2 := &mockHandler{}
		handler3 := &mockHandler{}

		err := engine.RegisterHandler(StatusPending, handler1)
		require.NoError(t, err)

		err = engine.RegisterHandler(StatusBuilding, handler2)
		require.NoError(t, err)

		err = engine.RegisterHandler(StatusStarting, handler3)
		require.NoError(t, err)

		require.Equal(t, handler1, engine.handlers[StatusPending])
		require.Equal(t, handler2, engine.handlers[StatusBuilding])
		require.Equal(t, handler3, engine.handlers[StatusStarting])
	})
}

func TestEngine_SetTimeout(t *testing.T) {
	t.Parallel()

	t.Run("verify it delegates to TimeoutConfig", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		timeout := 5 * time.Minute
		engine.SetTimeout(StatusBuilding, timeout)

		// Verify timeout was set by retrieving it
		retrievedTimeout, ok := engine.timeouts.GetTimeout(StatusBuilding)
		require.True(t, ok)
		require.Equal(t, timeout, retrievedTimeout)
	})
}

func TestEngine_ProcessDeployment(t *testing.T) {
	t.Parallel()

	t.Run("terminal state handling - Active", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		deployment := repository.Deployment{
			ID:     "dep-1",
			Status: StatusActive.String(),
		}

		mockRepo.On("GetDeployment", mock.Anything, "dep-1").
			Return(deployment, nil)

		err := engine.ProcessDeployment(context.Background(), "dep-1")

		require.NoError(t, err)
		mockRepo.AssertNotCalled(t, "UpdateDeploymentStatus", mock.Anything, mock.Anything)
	})

	t.Run("terminal state handling - Failed", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		deployment := repository.Deployment{
			ID:     "dep-2",
			Status: StatusFailed.String(),
		}

		mockRepo.On("GetDeployment", mock.Anything, "dep-2").
			Return(deployment, nil)

		err := engine.ProcessDeployment(context.Background(), "dep-2")

		require.NoError(t, err)
	})

	t.Run("terminal state handling - Cancelled", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		deployment := repository.Deployment{
			ID:     "dep-3",
			Status: StatusCancelled.String(),
		}

		mockRepo.On("GetDeployment", mock.Anything, "dep-3").
			Return(deployment, nil)

		err := engine.ProcessDeployment(context.Background(), "dep-3")

		require.NoError(t, err)
	})

	t.Run("terminal state handling - Stopped", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		deployment := repository.Deployment{
			ID:     "dep-4",
			Status: StatusStopped.String(),
		}

		mockRepo.On("GetDeployment", mock.Anything, "dep-4").
			Return(deployment, nil)

		err := engine.ProcessDeployment(context.Background(), "dep-4")

		require.NoError(t, err)
	})

	t.Run("missing handler - no error, just return", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		deployment := repository.Deployment{
			ID:     "dep-5",
			Status: StatusPending.String(),
		}

		mockRepo.On("GetDeployment", mock.Anything, "dep-5").
			Return(deployment, nil)

		err := engine.ProcessDeployment(context.Background(), "dep-5")

		require.NoError(t, err)
	})

	t.Run("handler execution success path - transitions to OnSuccess status", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		deployment := repository.Deployment{
			ID:     "dep-6",
			AppID:  "app-1",
			Status: StatusPending.String(),
		}

		handler := &mockHandler{}
		handler.On("Handle", mock.Anything, &deployment).Return(nil)

		err := engine.RegisterHandler(StatusPending, handler)
		require.NoError(t, err)

		mockRepo.On("GetDeployment", mock.Anything, "dep-6").
			Return(deployment, nil)

		mockRepo.On("UpdateDeploymentStatus", mock.Anything, repository.UpdateDeploymentStatusParams{
			Status: StatusBuilding.String(),
			ID:     "dep-6",
		}).Return(nil)

		mockRepo.On("CreateDeploymentLog", mock.Anything, mock.MatchedBy(func(params repository.CreateDeploymentLogParams) bool {
			return params.DeploymentID == "dep-6" &&
				params.Event == StatusPending.String() &&
				params.Message != nil &&
				params.Error == nil
		})).Return(nil)

		err = engine.ProcessDeployment(context.Background(), "dep-6")

		require.NoError(t, err)
		handler.AssertExpectations(t)
	})

	t.Run("handler execution failure path - transitions to OnFailure status", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		deployment := repository.Deployment{
			ID:     "dep-7",
			AppID:  "app-1",
			Status: StatusPending.String(),
		}

		handlerErr := errors.New("build failed")
		handler := &mockHandler{}
		handler.On("Handle", mock.Anything, &deployment).Return(handlerErr)

		err := engine.RegisterHandler(StatusPending, handler)
		require.NoError(t, err)

		mockRepo.On("GetDeployment", mock.Anything, "dep-7").
			Return(deployment, nil)

		mockRepo.On("UpdateDeploymentStatusWithStoppedAt", mock.Anything, repository.UpdateDeploymentStatusWithStoppedAtParams{
			Status: StatusFailed.String(),
			ID:     "dep-7",
		}).Return(nil)

		mockRepo.On("CreateDeploymentLog", mock.Anything, mock.MatchedBy(func(params repository.CreateDeploymentLogParams) bool {
			return params.DeploymentID == "dep-7" &&
				params.Event == StatusPending.String() &&
				params.Message == nil &&
				params.Error != nil
		})).Return(nil)

		err = engine.ProcessDeployment(context.Background(), "dep-7")

		require.NoError(t, err)
		handler.AssertExpectations(t)
	})

	t.Run("GetDeployment error handling", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		expectedErr := errors.New("database error")
		mockRepo.On("GetDeployment", mock.Anything, "dep-8").
			Return(repository.Deployment{}, expectedErr)

		err := engine.ProcessDeployment(context.Background(), "dep-8")

		require.Error(t, err)
		require.Equal(t, expectedErr, err)
	})

	t.Run("status update Active with UpdateDeploymentStatusWithActivatedAt", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		deployment := repository.Deployment{
			ID:     "dep-9",
			AppID:  "app-1",
			Status: StatusActivating.String(),
		}

		handler := &mockHandler{}
		handler.On("Handle", mock.Anything, &deployment).Return(nil)

		err := engine.RegisterHandler(StatusActivating, handler)
		require.NoError(t, err)

		mockRepo.On("GetDeployment", mock.Anything, "dep-9").
			Return(deployment, nil)

		mockRepo.On("UpdateDeploymentStatusWithActivatedAt", mock.Anything, repository.UpdateDeploymentStatusWithActivatedAtParams{
			Status: StatusActive.String(),
			ID:     "dep-9",
		}).Return(nil)

		mockRepo.On("CreateDeploymentLog", mock.Anything, mock.Anything).Return(nil)

		err = engine.ProcessDeployment(context.Background(), "dep-9")

		require.NoError(t, err)
		handler.AssertExpectations(t)
	})

	t.Run("status update Stopped with UpdateDeploymentStatusWithStoppedAt", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		deployment := repository.Deployment{
			ID:     "dep-10",
			AppID:  "app-1",
			Status: StatusReadyToShutdown.String(),
		}

		handler := &mockHandler{}
		handler.On("Handle", mock.Anything, &deployment).Return(nil)

		err := engine.RegisterHandler(StatusReadyToShutdown, handler)
		require.NoError(t, err)

		mockRepo.On("GetDeployment", mock.Anything, "dep-10").
			Return(deployment, nil)

		mockRepo.On("UpdateDeploymentStatusWithStoppedAt", mock.Anything, repository.UpdateDeploymentStatusWithStoppedAtParams{
			Status: StatusStopped.String(),
			ID:     "dep-10",
		}).Return(nil)

		mockRepo.On("CreateDeploymentLog", mock.Anything, mock.Anything).Return(nil)

		err = engine.ProcessDeployment(context.Background(), "dep-10")

		require.NoError(t, err)
		handler.AssertExpectations(t)
	})

	t.Run("status update Cancelled with UpdateDeploymentStatusWithStoppedAt", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		deployment := repository.Deployment{
			ID:     "dep-11",
			AppID:  "app-1",
			Status: StatusCancelling.String(),
		}

		handler := &mockHandler{}
		handler.On("Handle", mock.Anything, &deployment).Return(nil)

		err := engine.RegisterHandler(StatusCancelling, handler)
		require.NoError(t, err)

		mockRepo.On("GetDeployment", mock.Anything, "dep-11").
			Return(deployment, nil)

		mockRepo.On("UpdateDeploymentStatusWithStoppedAt", mock.Anything, repository.UpdateDeploymentStatusWithStoppedAtParams{
			Status: StatusCancelled.String(),
			ID:     "dep-11",
		}).Return(nil)

		mockRepo.On("CreateDeploymentLog", mock.Anything, mock.Anything).Return(nil)

		err = engine.ProcessDeployment(context.Background(), "dep-11")

		require.NoError(t, err)
		handler.AssertExpectations(t)
	})

	t.Run("status update other statuses with UpdateDeploymentStatus", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		deployment := repository.Deployment{
			ID:     "dep-12",
			AppID:  "app-1",
			Status: StatusBuilding.String(),
		}

		handler := &mockHandler{}
		handler.On("Handle", mock.Anything, &deployment).Return(nil)

		err := engine.RegisterHandler(StatusBuilding, handler)
		require.NoError(t, err)

		mockRepo.On("GetDeployment", mock.Anything, "dep-12").
			Return(deployment, nil)

		mockRepo.On("UpdateDeploymentStatus", mock.Anything, repository.UpdateDeploymentStatusParams{
			Status: StatusStarting.String(),
			ID:     "dep-12",
		}).Return(nil)

		mockRepo.On("CreateDeploymentLog", mock.Anything, mock.Anything).Return(nil)

		err = engine.ProcessDeployment(context.Background(), "dep-12")

		require.NoError(t, err)
		handler.AssertExpectations(t)
	})

	t.Run("log creation on success", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		deployment := repository.Deployment{
			ID:     "dep-13",
			AppID:  "app-1",
			Status: StatusPending.String(),
		}

		handler := &mockHandler{}
		handler.On("Handle", mock.Anything, &deployment).Return(nil)

		err := engine.RegisterHandler(StatusPending, handler)
		require.NoError(t, err)

		mockRepo.On("GetDeployment", mock.Anything, "dep-13").
			Return(deployment, nil)

		mockRepo.On("UpdateDeploymentStatus", mock.Anything, mock.Anything).Return(nil)

		mockRepo.On("CreateDeploymentLog", mock.Anything, mock.MatchedBy(func(params repository.CreateDeploymentLogParams) bool {
			return params.DeploymentID == "dep-13" &&
				params.Event == StatusPending.String() &&
				params.Message != nil &&
				params.Error == nil
		})).Return(nil)

		err = engine.ProcessDeployment(context.Background(), "dep-13")

		require.NoError(t, err)
	})

	t.Run("log creation on error", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		deployment := repository.Deployment{
			ID:     "dep-14",
			AppID:  "app-1",
			Status: StatusPending.String(),
		}

		handlerErr := errors.New("something went wrong")
		handler := &mockHandler{}
		handler.On("Handle", mock.Anything, &deployment).Return(handlerErr)

		err := engine.RegisterHandler(StatusPending, handler)
		require.NoError(t, err)

		mockRepo.On("GetDeployment", mock.Anything, "dep-14").
			Return(deployment, nil)

		mockRepo.On("UpdateDeploymentStatusWithStoppedAt", mock.Anything, mock.Anything).Return(nil)

		mockRepo.On("CreateDeploymentLog", mock.Anything, mock.MatchedBy(func(params repository.CreateDeploymentLogParams) bool {
			return params.DeploymentID == "dep-14" &&
				params.Event == StatusPending.String() &&
				params.Message == nil &&
				params.Error != nil
		})).Return(nil)

		err = engine.ProcessDeployment(context.Background(), "dep-14")

		require.NoError(t, err)
	})

	t.Run("log creation failure returns error", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		deployment := repository.Deployment{
			ID:     "dep-15",
			AppID:  "app-1",
			Status: StatusPending.String(),
		}

		handler := &mockHandler{}
		handler.On("Handle", mock.Anything, &deployment).Return(nil)

		err := engine.RegisterHandler(StatusPending, handler)
		require.NoError(t, err)

		mockRepo.On("GetDeployment", mock.Anything, "dep-15").
			Return(deployment, nil)

		mockRepo.On("CreateDeploymentLog", mock.Anything, mock.Anything).
			Return(errors.New("log creation failed"))

		err = engine.ProcessDeployment(context.Background(), "dep-15")

		require.Error(t, err)
		require.Contains(t, err.Error(), "logging failed")
	})
}

func TestEngine_ProcessDeployment_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		deploymentID   string
		status         Status
		handlerError   error
		expectedStatus Status
		updateMethod   string
	}{
		{
			name:           "pending to building on success",
			deploymentID:   "dep-100",
			status:         StatusPending,
			handlerError:   nil,
			expectedStatus: StatusBuilding,
			updateMethod:   "UpdateDeploymentStatus",
		},
		{
			name:           "pending to failed on error",
			deploymentID:   "dep-101",
			status:         StatusPending,
			handlerError:   errors.New("build setup failed"),
			expectedStatus: StatusFailed,
			updateMethod:   "UpdateDeploymentStatusWithStoppedAt",
		},
		{
			name:           "building to starting on success",
			deploymentID:   "dep-102",
			status:         StatusBuilding,
			handlerError:   nil,
			expectedStatus: StatusStarting,
			updateMethod:   "UpdateDeploymentStatus",
		},
		{
			name:           "building to failed on error",
			deploymentID:   "dep-103",
			status:         StatusBuilding,
			handlerError:   errors.New("compilation failed"),
			expectedStatus: StatusFailed,
			updateMethod:   "UpdateDeploymentStatusWithStoppedAt",
		},
		{
			name:           "activating to active on success",
			deploymentID:   "dep-104",
			status:         StatusActivating,
			handlerError:   nil,
			expectedStatus: StatusActive,
			updateMethod:   "UpdateDeploymentStatusWithActivatedAt",
		},
		{
			name:           "cancelling to cancelled on success",
			deploymentID:   "dep-105",
			status:         StatusCancelling,
			handlerError:   nil,
			expectedStatus: StatusCancelled,
			updateMethod:   "UpdateDeploymentStatusWithStoppedAt",
		},
		{
			name:           "ready_to_shutdown to stopped on success",
			deploymentID:   "dep-106",
			status:         StatusReadyToShutdown,
			handlerError:   nil,
			expectedStatus: StatusStopped,
			updateMethod:   "UpdateDeploymentStatusWithStoppedAt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := mocks.NewQueries(t)
			engine := NewEngine(mockRepo, EngineConfig{
				PollInterval: time.Second,
			})

			deployment := repository.Deployment{
				ID:     tt.deploymentID,
				AppID:  "app-1",
				Status: tt.status.String(),
			}

			handler := &mockHandler{}
			handler.On("Handle", mock.Anything, &deployment).Return(tt.handlerError)

			err := engine.RegisterHandler(tt.status, handler)
			require.NoError(t, err)

			mockRepo.On("GetDeployment", mock.Anything, tt.deploymentID).
				Return(deployment, nil)

			switch tt.updateMethod {
			case "UpdateDeploymentStatus":
				mockRepo.On("UpdateDeploymentStatus", mock.Anything, repository.UpdateDeploymentStatusParams{
					Status: tt.expectedStatus.String(),
					ID:     tt.deploymentID,
				}).Return(nil)
			case "UpdateDeploymentStatusWithActivatedAt":
				mockRepo.On("UpdateDeploymentStatusWithActivatedAt", mock.Anything, repository.UpdateDeploymentStatusWithActivatedAtParams{
					Status: tt.expectedStatus.String(),
					ID:     tt.deploymentID,
				}).Return(nil)
			case "UpdateDeploymentStatusWithStoppedAt":
				mockRepo.On("UpdateDeploymentStatusWithStoppedAt", mock.Anything, repository.UpdateDeploymentStatusWithStoppedAtParams{
					Status: tt.expectedStatus.String(),
					ID:     tt.deploymentID,
				}).Return(nil)
			}

			mockRepo.On("CreateDeploymentLog", mock.Anything, mock.Anything).Return(nil)

			err = engine.ProcessDeployment(context.Background(), tt.deploymentID)

			require.NoError(t, err)
			handler.AssertExpectations(t)
		})
	}
}

func TestEngine_ProcessDeployment_ContextCancellation(t *testing.T) {
	t.Parallel()

	t.Run("handler respects context cancellation", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		deployment := repository.Deployment{
			ID:     "dep-200",
			AppID:  "app-1",
			Status: StatusPending.String(),
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		handler := &mockHandler{}
		handler.On("Handle", mock.Anything, &deployment).Return(context.Canceled)

		err := engine.RegisterHandler(StatusPending, handler)
		require.NoError(t, err)

		mockRepo.On("GetDeployment", mock.Anything, "dep-200").
			Return(deployment, nil)

		mockRepo.On("UpdateDeploymentStatusWithStoppedAt", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("CreateDeploymentLog", mock.Anything, mock.Anything).Return(nil)

		err = engine.ProcessDeployment(ctx, "dep-200")

		require.NoError(t, err)
	})

	t.Run("GetDeployment with cancelled context", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mockRepo.On("GetDeployment", mock.Anything, "dep-201").
			Return(repository.Deployment{}, context.Canceled)

		err := engine.ProcessDeployment(ctx, "dep-201")

		require.Error(t, err)
		require.Equal(t, context.Canceled, err)
	})
}

func TestEngine_Run(t *testing.T) {
	t.Parallel()

	t.Run("processes initial batch on start", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: 100 * time.Millisecond,
		})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return([]repository.Deployment{}, nil).Once()

		mockRepo.On("GetDeploymentsNeedingWork", mock.Anything).
			Return([]repository.Deployment{}, nil).Once()

		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		err := engine.Run(ctx)

		require.ErrorIs(t, err, context.Canceled)
		mockRepo.AssertExpectations(t)
	})

	t.Run("processes batches on ticker interval", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		pollInterval := 50 * time.Millisecond
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: pollInterval,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()

		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return([]repository.Deployment{}, nil).Maybe()

		mockRepo.On("GetDeploymentsNeedingWork", mock.Anything).
			Return([]repository.Deployment{}, nil).Maybe()

		err := engine.Run(ctx)

		require.Error(t, err)
		mockRepo.AssertCalled(t, "GetStaleDeployments", mock.Anything, mock.Anything)
		mockRepo.AssertCalled(t, "GetDeploymentsNeedingWork", mock.Anything)
	})

	t.Run("continues running despite batch processing errors", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: 50 * time.Millisecond,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
		defer cancel()

		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return(nil, errors.New("database error")).Maybe()

		err := engine.Run(ctx)

		require.Error(t, err)
		mockRepo.AssertCalled(t, "GetStaleDeployments", mock.Anything, mock.Anything)
	})

	t.Run("stops gracefully on context cancellation", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Hour,
		})

		ctx, cancel := context.WithCancel(context.Background())

		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return([]repository.Deployment{}, nil).Once()
		mockRepo.On("GetDeploymentsNeedingWork", mock.Anything).
			Return([]repository.Deployment{}, nil).Once()

		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		err := engine.Run(ctx)

		require.ErrorIs(t, err, context.Canceled)
		mockRepo.AssertExpectations(t)
	})
}

func TestEngine_handleTimeouts(t *testing.T) {
	t.Parallel()

	t.Run("marks timed-out deployment as failed", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		engine.SetTimeout(StatusBuilding, 5*time.Minute)

		oldTime := time.Now().Add(-10 * time.Minute)
		staleDeployment := repository.Deployment{
			ID:        "dep-timeout-1",
			Status:    StatusBuilding.String(),
			CreatedAt: oldTime,
		}

		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return([]repository.Deployment{staleDeployment}, nil)

		mockRepo.On("UpdateDeploymentStatusWithStoppedAt", mock.Anything, repository.UpdateDeploymentStatusWithStoppedAtParams{
			Status: StatusFailed.String(),
			ID:     "dep-timeout-1",
		}).Return(nil)

		mockRepo.On("CreateDeploymentLog", mock.Anything, mock.MatchedBy(func(params repository.CreateDeploymentLogParams) bool {
			return params.DeploymentID == "dep-timeout-1" &&
				params.Event == "timeout" &&
				params.Error != nil
		})).Return(nil)

		err := engine.handleTimeouts(context.Background())

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("does not timeout deployment within time limit", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		engine.SetTimeout(StatusBuilding, 10*time.Minute)

		recentTime := time.Now().Add(-5 * time.Minute)
		staleDeployment := repository.Deployment{
			ID:        "dep-timeout-2",
			Status:    StatusBuilding.String(),
			CreatedAt: recentTime,
		}

		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return([]repository.Deployment{staleDeployment}, nil)

		err := engine.handleTimeouts(context.Background())

		require.NoError(t, err)
		mockRepo.AssertNotCalled(t, "UpdateDeploymentStatusWithStoppedAt")
		mockRepo.AssertNotCalled(t, "CreateDeploymentLog")
	})

	t.Run("skips terminal states", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		oldTime := time.Now().Add(-1 * time.Hour)
		terminalDeployments := []repository.Deployment{
			{ID: "dep-1", Status: StatusActive.String(), CreatedAt: oldTime},
			{ID: "dep-2", Status: StatusFailed.String(), CreatedAt: oldTime},
			{ID: "dep-3", Status: StatusCancelled.String(), CreatedAt: oldTime},
			{ID: "dep-4", Status: StatusStopped.String(), CreatedAt: oldTime},
		}

		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return(terminalDeployments, nil)

		err := engine.handleTimeouts(context.Background())

		require.NoError(t, err)
		mockRepo.AssertNotCalled(t, "UpdateDeploymentStatusWithStoppedAt")
	})

	t.Run("skips deployments without configured timeout", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		oldTime := time.Now().Add(-1 * time.Hour)
		deployment := repository.Deployment{
			ID:        "dep-no-timeout",
			Status:    StatusPending.String(),
			CreatedAt: oldTime,
		}

		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return([]repository.Deployment{deployment}, nil)

		err := engine.handleTimeouts(context.Background())

		require.NoError(t, err)
		mockRepo.AssertNotCalled(t, "UpdateDeploymentStatusWithStoppedAt")
	})

	t.Run("continues processing after update failure", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		engine.SetTimeout(StatusBuilding, 5*time.Minute)

		oldTime := time.Now().Add(-10 * time.Minute)
		deployments := []repository.Deployment{
			{ID: "dep-1", Status: StatusBuilding.String(), CreatedAt: oldTime},
			{ID: "dep-2", Status: StatusBuilding.String(), CreatedAt: oldTime},
		}

		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return(deployments, nil)

		mockRepo.On("UpdateDeploymentStatusWithStoppedAt", mock.Anything, repository.UpdateDeploymentStatusWithStoppedAtParams{
			Status: StatusFailed.String(),
			ID:     "dep-1",
		}).Return(errors.New("update failed")).Once()

		mockRepo.On("UpdateDeploymentStatusWithStoppedAt", mock.Anything, repository.UpdateDeploymentStatusWithStoppedAtParams{
			Status: StatusFailed.String(),
			ID:     "dep-2",
		}).Return(nil).Once()

		mockRepo.On("CreateDeploymentLog", mock.Anything, mock.Anything).Return(nil)

		err := engine.handleTimeouts(context.Background())

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("handles GetStaleDeployments error", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		expectedErr := errors.New("database error")
		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return(nil, expectedErr)

		err := engine.handleTimeouts(context.Background())

		require.Error(t, err)
		require.Equal(t, expectedErr, err)
	})

	t.Run("uses configured timeout durations", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		customTimeout := 1 * time.Minute
		engine.SetTimeout(StatusBuilding, customTimeout)

		recentDep := repository.Deployment{
			ID:        "dep-recent",
			Status:    StatusBuilding.String(),
			CreatedAt: time.Now().Add(-30 * time.Second),
		}

		oldDep := repository.Deployment{
			ID:        "dep-old",
			Status:    StatusBuilding.String(),
			CreatedAt: time.Now().Add(-2 * time.Minute),
		}

		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return([]repository.Deployment{recentDep, oldDep}, nil)

		mockRepo.On("UpdateDeploymentStatusWithStoppedAt", mock.Anything, repository.UpdateDeploymentStatusWithStoppedAtParams{
			Status: StatusFailed.String(),
			ID:     "dep-old",
		}).Return(nil).Once()

		mockRepo.On("CreateDeploymentLog", mock.Anything, mock.MatchedBy(func(params repository.CreateDeploymentLogParams) bool {
			return params.DeploymentID == "dep-old"
		})).Return(nil)

		err := engine.handleTimeouts(context.Background())

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestEngine_processWorkBatch(t *testing.T) {
	t.Parallel()

	t.Run("processes all deployments in batch", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		handler := &mockHandler{}
		err := engine.RegisterHandler(StatusPending, handler)
		require.NoError(t, err)

		deployments := []repository.Deployment{
			{ID: "dep-1", Status: StatusPending.String()},
			{ID: "dep-2", Status: StatusPending.String()},
		}

		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return([]repository.Deployment{}, nil)

		mockRepo.On("GetDeploymentsNeedingWork", mock.Anything).
			Return(deployments, nil)

		for _, dep := range deployments {
			depCopy := dep
			handler.On("Handle", mock.Anything, &depCopy).Return(nil)
			mockRepo.On("GetDeployment", mock.Anything, dep.ID).Return(depCopy, nil)
			mockRepo.On("UpdateDeploymentStatus", mock.Anything, mock.Anything).Return(nil)
			mockRepo.On("CreateDeploymentLog", mock.Anything, mock.Anything).Return(nil)
		}

		err = engine.processWorkBatch(context.Background())

		require.NoError(t, err)
		handler.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("continues processing after individual deployment error", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		handler := &mockHandler{}
		err := engine.RegisterHandler(StatusPending, handler)
		require.NoError(t, err)

		deployments := []repository.Deployment{
			{ID: "dep-1", Status: StatusPending.String()},
			{ID: "dep-2", Status: StatusPending.String()},
		}

		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return([]repository.Deployment{}, nil)

		mockRepo.On("GetDeploymentsNeedingWork", mock.Anything).
			Return(deployments, nil)

		mockRepo.On("GetDeployment", mock.Anything, "dep-1").
			Return(repository.Deployment{}, errors.New("database error")).Once()

		dep2 := deployments[1]
		handler.On("Handle", mock.Anything, &dep2).Return(nil)
		mockRepo.On("GetDeployment", mock.Anything, "dep-2").Return(dep2, nil)
		mockRepo.On("UpdateDeploymentStatus", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("CreateDeploymentLog", mock.Anything, mock.Anything).Return(nil)

		err = engine.processWorkBatch(context.Background())

		require.NoError(t, err)
		handler.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns early if no deployments need work", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return([]repository.Deployment{}, nil)

		mockRepo.On("GetDeploymentsNeedingWork", mock.Anything).
			Return([]repository.Deployment{}, nil)

		err := engine.processWorkBatch(context.Background())

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("handles GetDeploymentsNeedingWork error", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return([]repository.Deployment{}, nil)

		expectedErr := errors.New("database error")
		mockRepo.On("GetDeploymentsNeedingWork", mock.Anything).
			Return(nil, expectedErr)

		err := engine.processWorkBatch(context.Background())

		require.Error(t, err)
		require.Equal(t, expectedErr, err)
	})

	t.Run("calls handleTimeouts first", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return([]repository.Deployment{}, nil).Once()

		mockRepo.On("GetDeploymentsNeedingWork", mock.Anything).
			Return([]repository.Deployment{}, nil).Once()

		err := engine.processWorkBatch(context.Background())

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error if handleTimeouts fails", func(t *testing.T) {
		t.Parallel()

		mockRepo := mocks.NewQueries(t)
		engine := NewEngine(mockRepo, EngineConfig{
			PollInterval: time.Second,
		})

		expectedErr := errors.New("timeout handling failed")
		mockRepo.On("GetStaleDeployments", mock.Anything, mock.Anything).
			Return(nil, expectedErr)

		err := engine.processWorkBatch(context.Background())

		require.Error(t, err)
		require.Equal(t, expectedErr, err)
		mockRepo.AssertNotCalled(t, "GetDeploymentsNeedingWork")
	})
}
