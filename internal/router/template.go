/*
   GoToSocial
   Copyright (C) 2021-2022 GoToSocial Authors admin@gotosocial.org

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package router

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/superseriousbusiness/gotosocial/internal/api/model"
	"github.com/superseriousbusiness/gotosocial/internal/config"
	"github.com/superseriousbusiness/gotosocial/internal/log"
	"github.com/superseriousbusiness/gotosocial/internal/regexes"
	"github.com/superseriousbusiness/gotosocial/internal/util"
)

const (
	justTime     = "15:04"
	dateYear     = "Jan 02, 2006"
	dateTime     = "Jan 02, 15:04"
	dateYearTime = "Jan 02, 2006, 15:04"
	monthYear    = "Jan, 2006"
	badTimestamp = "bad timestamp"
)

// LoadTemplates loads html templates for use by the given engine
func LoadTemplates(engine *gin.Engine) error {
	templateBaseDir := config.GetWebTemplateBaseDir()
	if templateBaseDir == "" {
		return fmt.Errorf("%s cannot be empty and must be a relative or absolute path", config.WebTemplateBaseDirFlag())
	}

	templateBaseDir, err := filepath.Abs(templateBaseDir)
	if err != nil {
		return fmt.Errorf("error getting absolute path of %s: %s", templateBaseDir, err)
	}

	if _, err := os.Stat(filepath.Join(templateBaseDir, "index.tmpl")); err != nil {
		return fmt.Errorf("%s doesn't seem to contain the templates; index.tmpl is missing: %w", templateBaseDir, err)
	}

	engine.LoadHTMLGlob(filepath.Join(templateBaseDir, "*"))
	return nil
}

func oddOrEven(n int) string {
	if n%2 == 0 {
		return "even"
	}
	return "odd"
}

func escape(str string) template.HTML {
	/* #nosec G203 */
	return template.HTML(template.HTMLEscapeString(str))
}

func noescape(str string) template.HTML {
	/* #nosec G203 */
	return template.HTML(str)
}

func noescapeAttr(str string) template.HTMLAttr {
	/* #nosec G203 */
	return template.HTMLAttr(str)
}

func timestamp(stamp string) string {
	t, err := util.ParseISO8601(stamp)
	if err != nil {
		log.Errorf("error parsing timestamp %s: %s", stamp, err)
		return badTimestamp
	}

	t = t.Local()

	tYear, tMonth, tDay := t.Date()
	now := time.Now()
	currentYear, currentMonth, currentDay := now.Date()

	switch {
	case tYear == currentYear && tMonth == currentMonth && tDay == currentDay:
		return "Today, " + t.Format(justTime)
	case tYear == currentYear:
		return t.Format(dateTime)
	default:
		return t.Format(dateYear)
	}
}

func timestampPrecise(stamp string) string {
	t, err := util.ParseISO8601(stamp)
	if err != nil {
		log.Errorf("error parsing timestamp %s: %s", stamp, err)
		return badTimestamp
	}
	return t.Local().Format(dateYearTime)
}

func timestampVague(stamp string) string {
	t, err := util.ParseISO8601(stamp)
	if err != nil {
		log.Errorf("error parsing timestamp %s: %s", stamp, err)
		return badTimestamp
	}
	return t.Format(monthYear)
}

type iconWithLabel struct {
	faIcon string
	label  string
}

func visibilityIcon(visibility model.Visibility) template.HTML {
	var icon iconWithLabel

	switch visibility {
	case model.VisibilityPublic:
		icon = iconWithLabel{"globe", "public"}
	case model.VisibilityUnlisted:
		icon = iconWithLabel{"unlock", "unlisted"}
	case model.VisibilityPrivate:
		icon = iconWithLabel{"lock", "private"}
	case model.VisibilityMutualsOnly:
		icon = iconWithLabel{"handshake-o", "mutuals only"}
	case model.VisibilityDirect:
		icon = iconWithLabel{"envelope", "direct"}
	}

	/* #nosec G203 */
	return template.HTML(fmt.Sprintf(`<i aria-label="Visibility: %v" class="fa fa-%v"></i>`, icon.label, icon.faIcon))
}

// replaces shortcodes in `text` with the emoji in `emojis`
// text is a template.HTML to affirm that the input of this function is already escaped
func emojify(emojis []model.Emoji, text template.HTML) template.HTML {
	emojisMap := make(map[string]model.Emoji, len(emojis))

	for _, emoji := range emojis {
		shortcode := ":" + emoji.Shortcode + ":"
		emojisMap[shortcode] = emoji
	}

	out := regexes.ReplaceAllStringFunc(
		regexes.EmojiFinder,
		string(text),
		func(shortcode string, buf *bytes.Buffer) string {
			// Look for emoji according to this shortcode
			emoji, ok := emojisMap[shortcode]
			if !ok {
				return shortcode
			}

			// Escape raw emoji content
			safeURL := html.EscapeString(emoji.URL)
			safeCode := html.EscapeString(emoji.Shortcode)

			// Write HTML emoji repr to buffer
			buf.WriteString(`<img src="`)
			buf.WriteString(safeURL)
			buf.WriteString(`" title=":`)
			buf.WriteString(safeCode)
			buf.WriteString(`:" alt=":`)
			buf.WriteString(safeCode)
			buf.WriteString(`:" class="emoji"/>`)

			return buf.String()
		},
	)

	/* #nosec G203 */
	// (this is escaped above)
	return template.HTML(out)
}

func LoadTemplateFunctions(engine *gin.Engine) {
	engine.SetFuncMap(template.FuncMap{
		"escape":           escape,
		"noescape":         noescape,
		"noescapeAttr":     noescapeAttr,
		"oddOrEven":        oddOrEven,
		"visibilityIcon":   visibilityIcon,
		"timestamp":        timestamp,
		"timestampVague":   timestampVague,
		"timestampPrecise": timestampPrecise,
		"emojify":          emojify,
	})
}
