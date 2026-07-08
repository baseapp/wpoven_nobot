package captcha

import (
	"crypto/rand"
	"fmt"
	"git.gammaspectra.live/git/go-away/lib/challenge"
	"git.gammaspectra.live/git/go-away/utils"
	"github.com/goccy/go-yaml/ast"
	"html/template"
	"math/big"
	unsaferand "math/rand/v2"
	"net/http"
	"strings"
	"time"
)

func init() {
	challenge.Runtimes[Key] = FillRegistration
}

const Key = "captcha"

func FillRegistration(state challenge.StateInterface, reg *challenge.Registration, parameters ast.Node) error {
	reg.Class = challenge.ClassBlocking
	reg.Verify = nil

	reg.IssueChallenge = func(w http.ResponseWriter, r *http.Request, key challenge.Key, expiry time.Time) challenge.VerifyResult {
		// Generate random captcha string
		captchaText := generateRandomText(5)

		// Store expected text in ChallengeMap as Result (which is signed and encrypted in client cookie)
		data := challenge.RequestDataFromContext(r.Context())
		data.IssueChallengeToken(reg, key, []byte(strings.ToLower(captchaText)), expiry, false)

		// Generate SVG representation
		svgContent := generateSVG(captchaText)

		// Action URL for the form submission
		verifyURI, err := challenge.RedirectUrl(r, reg)
		if err != nil {
			return challenge.VerifyResultFail
		}
		verifyURI.Path = reg.Path + "/verify"

		state.ChallengePage(w, r, state.Settings().ChallengeResponseCode, reg, map[string]any{
			"Title":      state.Strings().Get("title_captcha"),
			"CaptchaSVG": template.HTML(svgContent),
			"VerifyURL":  verifyURI.String(),
			"Error":      "",
		})

		return challenge.VerifyResultNone
	}

	mux := http.NewServeMux()

	// POST verification route
	mux.HandleFunc("POST "+reg.Path+"/verify", func(w http.ResponseWriter, r *http.Request) {
		data := challenge.RequestDataFromContext(r.Context())

		// Read input
		userInput := strings.TrimSpace(strings.ToLower(r.FormValue("captcha_input")))

		// Read expected from ChallengeMap
		token, ok := data.ChallengeMap[reg.Name]
		if !ok || len(token.Result) == 0 {
			state.ErrorPage(w, r, http.StatusBadRequest, fmt.Errorf("challenge expired or invalid; please reload"), "/")
			return
		}

		expectedText := string(token.Result)

		if userInput != "" && userInput == expectedText {
			// Solved! Issue token as OK = true
			expiration := data.Expiration(reg.Duration)
			key := challenge.GetChallengeKeyForRequest(state, reg, expiration, r)
			data.IssueChallengeToken(reg, key, []byte(userInput), expiration, true)

			// Redirect back to the target URL safely
			redirectUrl := r.URL.Query().Get(challenge.QueryArgRedirect)
			if redirectUrl == "" {
				redirectUrl = r.URL.Query().Get(challenge.QueryArgReferer)
			}
			if redirectUrl == "" {
				redirectUrl = "/"
			}
			redirectUrl, err := utils.EnsureNoOpenRedirect(redirectUrl)
			if err != nil {
				redirectUrl = "/"
			}

			data.ResponseHeaders(w)
			http.Redirect(w, r, redirectUrl, http.StatusTemporaryRedirect)
		} else {
			// Incorrect answer! Generate a new captcha to prevent brute forcing
			captchaText := generateRandomText(5)
			expiration := data.Expiration(reg.Duration)
			key := challenge.GetChallengeKeyForRequest(state, reg, expiration, r)
			data.IssueChallengeToken(reg, key, []byte(strings.ToLower(captchaText)), expiration, false)

			// Write the response headers so the updated cookie with the new captcha text is saved
			data.ResponseHeaders(w)

			// Show the page again with an error
			svgContent := generateSVG(captchaText)

			verifyURI, err := challenge.RedirectUrl(r, reg)
			if err != nil {
				state.ErrorPage(w, r, http.StatusInternalServerError, err, "")
				return
			}
			verifyURI.Path = reg.Path + "/verify"

			state.ChallengePage(w, r, state.Settings().ChallengeResponseCode, reg, map[string]any{
				"Title":      state.Strings().Get("title_captcha"),
				"CaptchaSVG": template.HTML(svgContent),
				"VerifyURL":  verifyURI.String(),
				"Error":      state.Strings().Get("captcha_error_incorrect"),
			})
		}
	})

	reg.Handler = mux

	return nil
}

func generateRandomText(length int) string {
	chars := "23456789abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ"
	var sb strings.Builder
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			sb.WriteByte(chars[unsaferand.IntN(len(chars))])
		} else {
			sb.WriteByte(chars[n.Int64()])
		}
	}
	return sb.String()
}

func generateSVG(text string) string {
	var sb strings.Builder
	width := 240
	height := 80

	sb.WriteString(fmt.Sprintf(`<svg width="%d" height="%d" viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg">`, width, height, width, height))

	// Background with a nice gradient
	sb.WriteString(`<defs>`)
	sb.WriteString(`<linearGradient id="bgGrad" x1="0%" y1="0%" x2="100%" y2="100%">`)
	sb.WriteString(`<stop offset="0%" style="stop-color:#1a1c24;stop-opacity:1" />`)
	sb.WriteString(`<stop offset="100%" style="stop-color:#0d0e12;stop-opacity:1" />`)
	sb.WriteString(`</linearGradient>`)
	sb.WriteString(`</defs>`)
	sb.WriteString(`<rect width="100%" height="100%" fill="url(#bgGrad)" rx="8" />`)

	// Background noise: Grid or lines
	numLines := 6
	for i := 0; i < numLines; i++ {
		x1 := unsaferand.IntN(width)
		y1 := unsaferand.IntN(height)
		x2 := unsaferand.IntN(width)
		y2 := unsaferand.IntN(height)
		strokeColor := getRandomColor(0.3)
		strokeWidth := 1.5 + unsaferand.Float64()*1.5
		sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="%.2f" />`, x1, y1, x2, y2, strokeColor, strokeWidth))
	}

	// Noise: Circles
	numCircles := 12
	for i := 0; i < numCircles; i++ {
		cx := unsaferand.IntN(width)
		cy := unsaferand.IntN(height)
		r := 2 + unsaferand.IntN(4)
		fillColor := getRandomColor(0.2)
		sb.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="%d" fill="%s" />`, cx, cy, r, fillColor))
	}

	type charElement struct {
		x        int
		y        int
		char     string
		fontSize int
		angle    int
		color    string
		isFake   bool
	}

	var elements []charElement

	// Add real characters
	charWidth := width / (len(text) + 1)
	for i, char := range text {
		x := 20 + i*charWidth + unsaferand.IntN(10)
		y := 45 + unsaferand.IntN(15)
		fontSize := 32 + unsaferand.IntN(10)
		angle := -15 + unsaferand.IntN(30)
		color := getRandomColor(1.0)

		elements = append(elements, charElement{
			x:        x,
			y:        y,
			char:     string(char),
			fontSize: fontSize,
			angle:    angle,
			color:    color,
			isFake:   false,
		})
	}

	// Add 3 distractor (fake) characters with opacity 0
	fakeChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for i := 0; i < 3; i++ {
		fakeChar := string(fakeChars[unsaferand.IntN(len(fakeChars))])
		x := unsaferand.IntN(width)
		y := unsaferand.IntN(height)
		elements = append(elements, charElement{
			x:        x,
			y:        y,
			char:     fakeChar,
			fontSize: 1 + unsaferand.IntN(5),
			angle:    unsaferand.IntN(360),
			color:    "transparent",
			isFake:   true,
		})
	}

	// Shuffle elements to scramble DOM order
	unsaferand.Shuffle(len(elements), func(i, j int) {
		elements[i], elements[j] = elements[j], elements[i]
	})

	// Render elements
	for _, el := range elements {
		if el.isFake {
			sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-size="0" opacity="0" fill="none" style="display:none;visibility:hidden;">%s</text>`, el.x, el.y, el.char))
		} else {
			sb.WriteString(fmt.Sprintf(
				`<text x="%d" y="%d" font-family="'Inter', 'Segoe UI', sans-serif" font-weight="bold" font-size="%d" fill="%s" transform="rotate(%d, %d, %d)">%s</text>`,
				el.x, el.y, el.fontSize, el.color, el.angle, el.x, el.y, el.char,
			))
		}
	}

	sb.WriteString(`</svg>`)
	return sb.String()
}

func getRandomColor(alpha float64) string {
	h := unsaferand.IntN(360)
	s := 70 + unsaferand.IntN(30)
	l := 50 + unsaferand.IntN(20)
	return fmt.Sprintf("hsla(%d, %d%%, %d%%, %.2f)", h, s, l, alpha)
}
