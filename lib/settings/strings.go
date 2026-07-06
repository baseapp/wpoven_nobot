package settings

import (
	"git.gammaspectra.live/git/go-away/utils"
)

var DefaultStrings = utils.NewStrings(map[string]string{
	"title_challenge": "Checking you are not a bot",
	"title_error":     "Oh no!",

	"noscript_warning": "<p>Sadly, you may need to enable JavaScript to get past this challenge. This is required because AI companies have changed the social contract around how website hosting works.</p>",

	"details_title": "Why am I seeing this?",
	"details_text": `
<p>
    You are seeing this page because this website is protected by a Web Application Firewall (WAF) 
    designed to mitigate malicious scanning, automated vulnerability probing, and malware distribution.
</p>
<p>
    Automated mass scanning and aggressive request volumes can degrade server performance, 
    causing downtime and making resources inaccessible for legitimate visitors.
</p>
<p>
    <strong>Note:</strong> Security verification challenges require modern JavaScript features. 
    If you are experiencing issues, please ensure security plugins that disable JavaScript 
    (such as JShelter) are temporarily paused or whitelisted for this domain.
</p>
`,
	"details_contact_admin_with_request_id": "If you have any issues contact the site administrator and provide the following Request Id",

	"button_refresh_page": "Refresh page",

	"status_loading_challenge":   "Loading challenge",
	"status_starting_challenge":  "Starting challenge",
	"status_loading":             "Loading...",
	"status_calculating":         "Calculating...",
	"status_challenge_success":   "Challenge success!",
	"status_challenge_done_took": "Done! Took",
	"status_error":               "Error:",
})
