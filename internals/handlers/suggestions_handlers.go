package handlers

import (
	"net/http"
	"strings"

	"github.com/andiq123/FindVibeFiber/internals/core/ports"
	"github.com/gofiber/fiber/v3"
)

type SuggestionsHandlers struct {
	suggestionsService ports.ISuggestionsService
}

var _ ports.ISuggestionsHandlers = (*SuggestionsHandlers)(nil)

func NewSuggestionsHandlers(suggestionsService ports.ISuggestionsService) *SuggestionsHandlers {
	return &SuggestionsHandlers{
		suggestionsService: suggestionsService,
	}
}

func (s *SuggestionsHandlers) GetSuggestions(c fiber.Ctx) error {
	query := strings.ReplaceAll(c.Query("q"), " ", "+")

	resultChan := make(chan []string)
	errorChan := make(chan error)

	go func() {
		defer close(resultChan)
		defer close(errorChan)
		suggestions, err := s.suggestionsService.GetSuggestions(query)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- suggestions
	}()

	var suggestions []string
	select {
	case res := <-resultChan:
		suggestions = res
	case err := <-errorChan:
		return c.Status(http.StatusInternalServerError).JSON(map[string]string{"error": err.Error()})
	}

	return c.Status(http.StatusOK).JSON(suggestions)
}
