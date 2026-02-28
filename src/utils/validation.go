package utils

import (
	"errors"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"regexp"
	"ricehub/src/errs"
	"slices"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	enLocales "github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	"go.uber.org/zap"
)

var translator ut.Translator

func addCustomTag(v *validator.Validate, tag string, validate func(field string) bool, translation string) {
	v.RegisterValidation(tag, func(fl validator.FieldLevel) bool {
		fieldStr := fl.Field().String()
		return validate(fieldStr)
	})

	v.RegisterTranslation(tag, translator, func(ut ut.Translator) error {
		return ut.Add(tag, translation, true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T(tag, fe.Field())
		return t
	})
}

func InitValidator() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// setup translation
		en := enLocales.New()
		uni := ut.New(en, en)

		translator, _ = uni.GetTranslator("en")
		enTranslations.RegisterDefaultTranslations(v, translator)

		// custom validation tags
		addCustomTag(v, "displayname", func(displayName string) bool {
			re := regexp.MustCompile(`^[a-zA-Z0-9 _\-.]+$`)
			return re.MatchString(displayName)
		}, "{0} can contain only a-Z, 0-9, whitespace, dot, underscore and dash characters.")

		addCustomTag(v, "ricetitle", func(riceTitle string) bool {
			re := regexp.MustCompile(`^[\[\]()a-zA-Z0-9 '_-]+$`)
			return re.MatchString(riceTitle)
		}, "{0} can contain only a-Z, 0-9, -, _, [], () and whitespace characters.")

		zap.L().Info("Validator initialized")
	}
}

func checkValidationErrors(err error) error {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		translated := slices.Collect(maps.Values(ve.Translate(translator)))
		return errs.UserErrors(translated, http.StatusBadRequest)
	} else if errors.Is(err, io.EOF) {
		return errs.UserError("Request body is required", http.StatusBadRequest)
	}

	return errs.UserError("Failed to parse and decode request body", http.StatusBadRequest)
}

func ValidateJSON(c *gin.Context, obj any) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return checkValidationErrors(err)
	}
	return nil
}

func ValidateForm(c *gin.Context, obj any) error {
	if err := c.ShouldBind(obj); err != nil {
		return checkValidationErrors(err)
	}
	return nil
}

var openFailed = errs.UserError("Couldn't open and read the uploaded file", http.StatusUnprocessableEntity)

func ValidateFileAsImage(formFile *multipart.FileHeader) (string, error) {
	// file, err := formFile.Open()
	// if err != nil {
	// 	return "", openFailed
	// }

	// mtype, _ := mimetype.DetectReader(file)
	// log.Println(mtype)
	// if !mtype.Is("image/jpeg") && !mtype.Is("image/png") {
	// 	return "", errs.UserError("Unsupported file type! Only png/jpeg is accepted", http.StatusUnsupportedMediaType)
	// }

	// return mtype.Extension(), nil
	name := formFile.Filename
	if strings.HasSuffix(name, ".png") {
		return ".png", nil
	} else if strings.HasSuffix(name, ".jpg") {
		return ".jpg", nil
	} else {
		return "", errs.UserError("Unsupported file type! Only png/jpeg is accepted", http.StatusUnsupportedMediaType)
	}
}

func ValidateFileAsArchive(formFile *multipart.FileHeader) (string, error) {
	file, err := formFile.Open()
	if err != nil {
		return "", openFailed
	}

	mtype, _ := mimetype.DetectReader(file)
	if !mtype.Is("application/zip") {
		return "", errs.UserError("Unsupported file type! Only zip is accepted", http.StatusUnsupportedMediaType)
	}

	return mtype.Extension(), nil
}

// case-insensitive version of strings.Contains
func containsI(text string, substr string) bool {
	return strings.Contains(
		strings.ToLower(text),
		strings.ToLower(substr),
	)
}

func IsUsernameBlacklisted(username string) bool {
	bl := Config.Blacklist

	// check for exact matches
	exact := append(bl.DisplayNames, bl.Usernames...)
	for _, word := range exact {
		if strings.EqualFold(word, username) {
			return true
		}
	}

	// check if contains
	for _, word := range bl.Words {
		if containsI(username, word) {
			return true
		}
	}

	return false
}

func IsDisplayNameBlacklisted(displayName string) bool {
	bl := Config.Blacklist

	contains := append(bl.Words, bl.DisplayNames...)
	for _, word := range contains {
		if containsI(displayName, word) {
			return true
		}
	}

	return false
}
