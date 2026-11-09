package core

import "errors"

// Sentinel errors returned by the core API. Callers should use errors.Is to
// match them, since every error is wrapped with contextual information.
var (
	// ErrInvalidVideoID indicates that the supplied video identifier does not
	// match the YouTube video ID format.
	ErrInvalidVideoID = errors.New("ID de vídeo inválido")

	// ErrMissingAPIKey indicates that YOUTUBE_API_KEY is not configured.
	ErrMissingAPIKey = errors.New("chave da API do YouTube não informada")

	// ErrVideoNotFound indicates that the YouTube Data API does not know the
	// requested video (empty result or HTTP 404).
	ErrVideoNotFound = errors.New("vídeo não encontrado")

	// ErrMetadataUnavailable indicates a metadata/API/network failure.
	ErrMetadataUnavailable = errors.New("metadados indisponíveis")

	// ErrTranscriptUnavailable indicates that every transcript strategy failed.
	ErrTranscriptUnavailable = errors.New("transcrição indisponível")
)
