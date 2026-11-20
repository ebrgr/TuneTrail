package cfg

import (
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ProjectURL is the public contact sent in the User-Agent to external services.
const ProjectURL = "https://github.com/ebrgr/TuneTrail"

// DeprecationNotices returns the deprecated-variable notices found at startup.
// Load runs before the logger exists, so initLogger reports them once.
func DeprecationNotices() []string {
	lock.RLock()
	defer lock.RUnlock()
	return append([]string(nil), startupNotices...)
}

func UserAgent() string {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.userAgent
}

func ListenAddr() string {
	lock.RLock()
	defer lock.RUnlock()
	// JoinHostPort brackets IPv6 literals (Koito #74); accept an already bracketed value.
	host := globalConfig.bindAddr
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}
	return net.JoinHostPort(host, strconv.Itoa(globalConfig.listenPort))
}

func ConfigDir() string {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.configDir
}

func SqliteEnabled() bool {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.sqliteEnabled
}

func DatabaseUrl() string {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.databaseUrl
}

func MusicBrainzUrl() string {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.musicBrainzUrl
}

func MusicBrainzRateLimit() int {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.musicBrainzRateLimit
}

func LogLevel() int {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.logLevel
}

func StructuredLogging() bool {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.structuredLogging
}

func LbzRelayEnabled() bool {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.lbzRelayEnabled
}

func LbzRelayUrl() string {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.lbzRelayUrl
}

func LbzRelayToken() string {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.lbzRelayToken
}

func DefaultPassword() string {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.defaultPw
}

func DefaultUsername() string {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.defaultUsername
}

func DefaultTheme() string {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.defaultTheme
}

func DeezerDisabled() bool {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.disableDeezer
}

func CoverArtArchiveDisabled() bool {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.disableCAA
}

func MusicBrainzDisabled() bool {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.disableMusicBrainz
}

func SubsonicEnabled() bool {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.subsonicEnabled
}

func SubsonicUrl() string {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.subsonicUrl
}

func SubsonicParams() string {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.subsonicParams
}

func LastFMApiKey() string {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.lastfmApiKey
}

func SkipImport() bool {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.skipImport
}

func AllowedHosts() []string {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.allowedHosts
}

func AllowAllHosts() bool {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.allowAllHosts
}

func AllowedOrigins() []string {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.allowedOrigins
}

func RateLimitDisabled() bool {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.disableRateLimit
}

func ThrottleImportMs() int {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.importThrottleMs
}

// returns the before, after times, in that order
func ImportWindow() (time.Time, time.Time) {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.importBefore, globalConfig.importAfter
}

func FetchImagesDuringImport() bool {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.fetchImageDuringImport
}

func ArtistSeparators() []*regexp.Regexp {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.artistSeparators
}

func LoginGate() bool {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.loginGate
}

func ForceTZ() *time.Location {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.forceTZ
}

func WebUI() string { lock.RLock(); defer lock.RUnlock(); return globalConfig.webUI }

// ActiveIngestEnabled returns the master switch loaded at startup.
func ActiveIngestEnabled() bool {
	lock.RLock()
	defer lock.RUnlock()
	return globalConfig.ingestActive
}
