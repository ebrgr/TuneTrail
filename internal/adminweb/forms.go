package adminweb

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
)

// FormLimits is supplied per action; upload size includes the multipart envelope.
type FormLimits struct {
	MaxBytes int64
	TempDir  string
}
type Upload struct {
	Filename, Path string
	Size           int64
}

// PreparedForm is returned only after EOF and all duplicate/limit checks have passed.
type PreparedForm struct {
	Values url.Values
	Files  map[string]Upload
}

// Close removes every staged file, including after an operation fails.
func (f *PreparedForm) Close() {
	for _, upload := range f.Files {
		_ = os.Remove(upload.Path)
	}
}

// ParseSessionForm requires a fresh session resolution by the caller before validating CSRF.
func (c *CSRF) ParseSessionForm(w http.ResponseWriter, r *http.Request, identity admin.Identity, limits FormLimits) (*PreparedForm, error) {
	binding, err := sessionBinding(identity)
	if err != nil {
		return nil, err
	}
	return c.parseForm(w, r, limits, func(token string) bool { return c.valid(token, "session-form", binding) })
}

// ParseLoginForm accepts only the signed preauth cookie and login-purpose token.
func (c *CSRF) ParseLoginForm(w http.ResponseWriter, r *http.Request, limits FormLimits) (*PreparedForm, error) {
	binding, _, ok := c.loginBinding(r)
	if !ok {
		return nil, ErrCSRF
	}
	return c.parseForm(w, r, limits, func(token string) bool { return c.valid(token, "login-form", binding) })
}

// formReadError maps an exceeded HTTP envelope without exposing parser or filesystem details.
func formReadError(err error) error {
	var disk *os.PathError
	if errors.As(err, &disk) {
		return err
	}
	var max *http.MaxBytesError
	if errors.As(err, &max) {
		return ErrTooLarge
	}
	return db.ErrInvalidInput
}

// parseForm never invokes a mutation and never spools a file before the leading CSRF token is valid.
func (c *CSRF) parseForm(w http.ResponseWriter, r *http.Request, limits FormLimits, validate func(string) bool) (form *PreparedForm, err error) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != "POST" {
		return nil, ErrMethod
	}
	if r.ContentLength > limits.MaxBytes && limits.MaxBytes > 0 {
		return nil, ErrTooLarge
	}
	if limits.MaxBytes <= 0 {
		return nil, db.ErrInvalidInput
	}
	// Body fields are the sole token source; query/header tokens are never a fallback.
	if r.URL.Query().Has("csrf_token") || len(r.Header.Values("X-CSRF-Token")) != 0 {
		return nil, ErrCSRF
	}
	r.Body = http.MaxBytesReader(w, r.Body, limits.MaxBytes)
	media, params, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if e != nil {
		return nil, db.ErrInvalidInput
	}
	f := &PreparedForm{Values: url.Values{}, Files: map[string]Upload{}}
	defer func() {
		if err != nil {
			f.Close()
		}
	}()
	switch media {
	case "application/x-www-form-urlencoded":
		data, e := io.ReadAll(io.LimitReader(r.Body, (64<<10)+1))
		if e != nil {
			return nil, formReadError(e)
		}
		if len(data) > 64<<10 {
			return nil, ErrTooLarge
		}
		f.Values, e = url.ParseQuery(string(data))
		if e != nil {
			return nil, db.ErrInvalidInput
		}
		if len(f.Values) > 32 {
			return nil, ErrTooLarge
		}
		for name, values := range f.Values {
			if len(name) > 100 || len(values) != 1 || len(values[0]) > 4096 {
				return nil, db.ErrInvalidInput
			}
		}
		if !validate(f.Values.Get("csrf_token")) {
			return nil, ErrCSRF
		}
	case "multipart/form-data":
		if params["boundary"] == "" {
			return nil, db.ErrInvalidInput
		}
		reader, e := r.MultipartReader()
		if e != nil {
			return nil, db.ErrInvalidInput
		}
		count := 0
		seen := map[string]bool{}
		for {
			part, e := reader.NextPart()
			if e == io.EOF {
				break
			}
			if e != nil {
				return nil, formReadError(e)
			}
			count++
			name := part.FormName()
			if count > 32 {
				return nil, ErrTooLarge
			}
			if seen[name] && name == "csrf_token" {
				return nil, ErrCSRF
			}
			if name == "" || len(name) > 100 || seen[name] {
				return nil, db.ErrInvalidInput
			}
			seen[name] = true
			if count == 1 {
				if name != "csrf_token" || part.FileName() != "" {
					return nil, ErrCSRF
				}
				token, e := io.ReadAll(io.LimitReader(part, 513))
				if e != nil {
					return nil, formReadError(e)
				}
				if len(token) > 512 || !validate(string(token)) {
					return nil, ErrCSRF
				}
				if e := part.Close(); e != nil {
					return nil, formReadError(e)
				}
				f.Values.Set(name, string(token))
				continue
			}
			if name == "csrf_token" {
				return nil, ErrCSRF
			}
			if filename := part.FileName(); filename != "" {
				file, e := os.CreateTemp(limits.TempDir, "koito-form-*")
				if e != nil {
					return nil, e
				}
				upload := Upload{Filename: filepath.Base(filename), Path: file.Name()}
				f.Files[name] = upload
				n, copyErr := io.Copy(file, part)
				closeErr := file.Close()
				if copyErr != nil {
					return nil, formReadError(copyErr)
				}
				if closeErr != nil {
					return nil, closeErr
				}
				upload.Size = n
				f.Files[name] = upload
			} else {
				value, e := io.ReadAll(io.LimitReader(part, 4097))
				if e != nil {
					return nil, formReadError(e)
				}
				if len(value) > 4096 {
					return nil, ErrTooLarge
				}
				f.Values.Set(name, string(value))
			}
			if e := part.Close(); e != nil {
				return nil, formReadError(e)
			}
		}
		if count == 0 {
			return nil, ErrCSRF
		}
		if _, e := io.Copy(io.Discard, r.Body); e != nil {
			return nil, formReadError(e)
		}
	default:
		return nil, db.ErrInvalidInput
	}
	// Credentials and action-specific fields are validated by their service after complete parsing.
	return f, nil
}

// SingleValue gives adapters an explicit body-only field contract.
func (f *PreparedForm) SingleValue(name string) string { return f.Values.Get(name) }
