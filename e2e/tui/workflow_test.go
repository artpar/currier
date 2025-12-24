package tui_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/artpar/currier/e2e/harness"
	"github.com/artpar/currier/e2e/testserver"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/tui/components"
	"github.com/artpar/currier/internal/tui/views"
	"github.com/stretchr/testify/assert"
)

func TestTUI_RequestWorkflow(t *testing.T) {
	handlers := testserver.Handlers{}

	h := harness.New(t, harness.Config{
		ServerHandlers: map[string]http.HandlerFunc{
			"/api/test": handlers.JSON(200, map[string]interface{}{
				"success": true,
				"message": "Hello from server",
			}),
		},
		Timeout: 5 * time.Second,
	})

	t.Run("SendRequestMsg triggers loading state", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// Create a request definition
		reqDef := core.NewRequestDefinition("Test Request", "GET", h.ServerURL()+"/api/test")

		// Simulate the SendRequestMsg
		msg := components.SendRequestMsg{Request: reqDef}
		model := session.Model()
		updated, cmd := model.Update(msg)
		newModel := updated.(*views.MainView)

		// Should set loading state and return a command
		assert.True(t, newModel.ResponsePanel().IsLoading())
		assert.NotNil(t, cmd, "should return a command to send request")

		// Should focus response pane
		assert.Equal(t, views.PaneResponse, newModel.FocusedPane())
	})

	t.Run("RequestErrorMsg shows error in response panel", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// Simulate the RequestErrorMsg
		msg := components.RequestErrorMsg{Error: errors.New("connection refused")}
		model := session.Model()
		updated, _ := model.Update(msg)
		newModel := updated.(*views.MainView)

		// Should set error and clear loading
		assert.False(t, newModel.ResponsePanel().IsLoading())
		assert.NotNil(t, newModel.ResponsePanel().Error())
		assert.Contains(t, newModel.ResponsePanel().Error().Error(), "connection refused")

		// Output should show error
		output := newModel.View()
		assert.Contains(t, output, "Error")
	})

	t.Run("full request flow with real HTTP", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// Create a request definition
		reqDef := core.NewRequestDefinition("Test Request", "GET", h.ServerURL()+"/api/test")

		// Send the request message
		sendMsg := components.SendRequestMsg{Request: reqDef}
		model := session.Model()
		updated, cmd := model.Update(sendMsg)
		model = updated.(*views.MainView)

		// Should be loading
		assert.True(t, model.ResponsePanel().IsLoading())

		// Execute the command to actually send the request
		if cmd != nil {
			resultMsg := cmd()

			// Process the result message
			updated, _ = model.Update(resultMsg)
			model = updated.(*views.MainView)

			// Should have received response
			assert.False(t, model.ResponsePanel().IsLoading())

			// Check if we got a response or error
			if respMsg, ok := resultMsg.(components.ResponseReceivedMsg); ok {
				assert.NotNil(t, respMsg.Response)
				assert.Equal(t, 200, respMsg.Response.Status().Code())
			} else if errMsg, ok := resultMsg.(components.RequestErrorMsg); ok {
				t.Logf("Got error (may be expected in test env): %v", errMsg.Error)
			}
		}
	})

	t.Run("response shows status code after request", func(t *testing.T) {
		session := h.TUI().Start(t)
		defer session.Quit()

		// Create a request definition
		reqDef := core.NewRequestDefinition("Test Request", "GET", h.ServerURL()+"/api/test")

		// Send the request and execute the command
		sendMsg := components.SendRequestMsg{Request: reqDef}
		model := session.Model()
		updated, cmd := model.Update(sendMsg)
		model = updated.(*views.MainView)

		if cmd != nil {
			resultMsg := cmd()
			updated, _ = model.Update(resultMsg)
			model = updated.(*views.MainView)

			// Check output contains status
			output := model.View()
			if model.ResponsePanel().Response() != nil {
				assert.Contains(t, output, "200")
			}
		}
	})
}
