package notices

import "net/url"

const DocsURL = RepoURL + "/tree/main/docs"

func BugReportURL(version string) string {
	if version == "" {
		return RepoURL + "/issues/new"
	}
	q := url.Values{}
	q.Set("version", version)
	return RepoURL + "/issues/new?" + q.Encode()
}

func FeatureRequestURL() string {
	return RepoURL + "/issues/new"
}

func SupportRequestURL() string {
	body := "**What are you trying to do**\n\n" +
		"I want aghelm to support another coding CLI.\n\n" +
		"**What you have in mind**\n\n" +
		"CLI name:\nHow to launch it:\nResume / session flags (if any):\n\n" +
		"**Area**\nConfig and tool support\n"
	return RepoURL + "/issues/new?body=" + url.QueryEscape(body)
}
