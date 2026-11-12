package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
	"github.com/zendev-sh/goai/provider/anthropic"
	"github.com/zendev-sh/goai/provider/compat"
	"github.com/zendev-sh/goai/provider/google"
	"github.com/zendev-sh/goai/provider/ollama"

	"github.com/marcuth/social-media-copilot/internal/models"
)

// Provider kinds supported by the app. Most hosted LLM APIs (OpenAI, DeepSeek,
// Groq, OpenRouter, LM Studio...) speak the OpenAI-compatible protocol.
const (
	KindOpenAICompatible = "openai_compatible"
	KindAnthropic        = "anthropic"
	KindGoogle           = "google"
	KindOllama           = "ollama"
)

type goaiModel = provider.LanguageModel

// ProviderService manages the user's AI providers.
type ProviderService struct {
	DB *sql.DB
}

const providerColumns = `id, name, kind, base_url, api_key, model, is_default, created_at, updated_at`

func scanProvider(row scanner) (models.AIProvider, error) {
	var p models.AIProvider
	err := row.Scan(&p.ID, &p.Name, &p.Kind, &p.BaseURL, &p.APIKey, &p.Model, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

// ListProviders returns every configured provider, default first.
func (s *ProviderService) ListProviders() ([]models.AIProvider, error) {
	rows, err := s.DB.Query(`SELECT ` + providerColumns + ` FROM ai_providers ORDER BY is_default DESC, name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.AIProvider{}
	for rows.Next() {
		p, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func validateProvider(p *models.AIProvider) error {
	p.Name = strings.TrimSpace(p.Name)
	p.BaseURL = strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	p.APIKey = strings.TrimSpace(p.APIKey)
	p.Model = strings.TrimSpace(p.Model)
	if p.Name == "" {
		return errors.New("dê um nome ao provedor")
	}
	if p.Model == "" {
		return errors.New("informe o modelo")
	}
	switch p.Kind {
	case KindOpenAICompatible:
		if p.BaseURL == "" {
			return errors.New("informe a URL base (ex.: https://api.openai.com/v1)")
		}
	case KindAnthropic, KindGoogle:
		if p.APIKey == "" {
			return errors.New("informe a chave de API")
		}
	case KindOllama:
	default:
		return fmt.Errorf("tipo de provedor desconhecido: %s", p.Kind)
	}
	return nil
}

// SaveProvider creates (ID 0) or updates a provider. The first provider, or
// one saved with IsDefault, becomes the default.
func (s *ProviderService) SaveProvider(p models.AIProvider) (models.AIProvider, error) {
	if err := validateProvider(&p); err != nil {
		return p, err
	}
	now := nowStamp()
	tx, err := s.DB.Begin()
	if err != nil {
		return p, err
	}
	defer tx.Rollback()
	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM ai_providers WHERE id != ?`, p.ID).Scan(&count); err != nil {
		return p, err
	}
	if count == 0 {
		p.IsDefault = true
	}
	if p.IsDefault {
		if _, err := tx.Exec(`UPDATE ai_providers SET is_default = 0`); err != nil {
			return p, err
		}
	}
	if p.ID == 0 {
		res, err := tx.Exec(`INSERT INTO ai_providers (name, kind, base_url, api_key, model, is_default, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, p.Name, p.Kind, p.BaseURL, p.APIKey, p.Model, p.IsDefault, now, now)
		if err != nil {
			return p, err
		}
		p.ID, _ = res.LastInsertId()
		p.CreatedAt = now
	} else {
		if _, err := tx.Exec(`UPDATE ai_providers SET name = ?, kind = ?, base_url = ?, api_key = ?, model = ?,
			is_default = ?, updated_at = ? WHERE id = ?`,
			p.Name, p.Kind, p.BaseURL, p.APIKey, p.Model, p.IsDefault, now, p.ID); err != nil {
			return p, err
		}
	}
	p.UpdatedAt = now
	return p, tx.Commit()
}

// DeleteProvider removes a provider. If it was the default, another one takes its place.
func (s *ProviderService) DeleteProvider(id int64) error {
	if _, err := s.DB.Exec(`DELETE FROM ai_providers WHERE id = ?`, id); err != nil {
		return err
	}
	_, err := s.DB.Exec(`UPDATE ai_providers SET is_default = 1 WHERE id = (SELECT id FROM ai_providers ORDER BY id LIMIT 1)
		AND NOT EXISTS (SELECT 1 FROM ai_providers WHERE is_default = 1)`)
	return err
}

// SetDefaultProvider makes a provider the one used by the AI features.
func (s *ProviderService) SetDefaultProvider(id int64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE ai_providers SET is_default = (id = ?)`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// TestProvider sends a tiny prompt with the given (possibly unsaved) settings.
func (s *ProviderService) TestProvider(ctx context.Context, p models.AIProvider) (string, error) {
	if err := validateProvider(&p); err != nil {
		return "", err
	}
	model, err := buildModel(p)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	res, err := goai.GenerateText(ctx, model,
		goai.WithPrompt("Responda apenas com a palavra: OK"),
		goai.WithMaxOutputTokens(512),
		goai.WithMaxRetries(0),
	)
	if err != nil {
		return "", friendlyAIError(err)
	}
	return strings.TrimSpace(res.Text), nil
}

func (s *ProviderService) defaultProvider() (models.AIProvider, error) {
	p, err := scanProvider(s.DB.QueryRow(`SELECT ` + providerColumns + ` FROM ai_providers ORDER BY is_default DESC, id LIMIT 1`))
	if errors.Is(err, sql.ErrNoRows) {
		return p, errors.New("nenhum provedor de IA configurado: adicione um em Configurações")
	}
	return p, err
}

// buildModel turns a stored provider into a goai language model.
func buildModel(p models.AIProvider) (provider.LanguageModel, error) {
	switch p.Kind {
	case KindOpenAICompatible:
		key := p.APIKey
		if key == "" {
			key = "not-needed" // local servers (LM Studio, vLLM...) ignore the key
		}
		return compat.Chat(p.Model, compat.WithBaseURL(p.BaseURL), compat.WithAPIKey(key)), nil
	case KindAnthropic:
		opts := []anthropic.Option{anthropic.WithAPIKey(p.APIKey)}
		if p.BaseURL != "" {
			opts = append(opts, anthropic.WithBaseURL(p.BaseURL))
		}
		return anthropic.Chat(p.Model, opts...), nil
	case KindGoogle:
		opts := []google.Option{google.WithAPIKey(p.APIKey)}
		if p.BaseURL != "" {
			opts = append(opts, google.WithBaseURL(p.BaseURL))
		}
		return google.Chat(p.Model, opts...), nil
	case KindOllama:
		var opts []ollama.Option
		if p.BaseURL != "" {
			opts = append(opts, ollama.WithBaseURL(strings.TrimSuffix(p.BaseURL, "/v1")))
		}
		return ollama.Chat(p.Model, opts...), nil
	}
	return nil, fmt.Errorf("tipo de provedor desconhecido: %s", p.Kind)
}

func friendlyAIError(err error) error {
	if errors.Is(err, context.Canceled) {
		return errors.New("geração cancelada")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("o provedor demorou demais para responder")
	}
	msg := err.Error()
	low := strings.ToLower(msg)
	switch {
	case strings.Contains(low, "401") || strings.Contains(low, "unauthorized") || strings.Contains(low, "invalid api key"):
		return fmt.Errorf("chave de API recusada pelo provedor (%s)", msg)
	case strings.Contains(low, "connection refused") || strings.Contains(low, "no such host"):
		return fmt.Errorf("não foi possível conectar ao provedor: confira a URL e se o servidor está rodando (%s)", msg)
	case strings.Contains(low, "404") || strings.Contains(low, "model"):
		return fmt.Errorf("o provedor recusou o pedido: confira o nome do modelo (%s)", msg)
	}
	return err
}
