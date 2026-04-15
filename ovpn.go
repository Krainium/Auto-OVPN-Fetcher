package main

import (
        "bufio"
        "encoding/base64"
        "encoding/csv"
        "fmt"
        "io"
        "net/http"
        "os"
        "path/filepath"
        "sort"
        "strconv"
        "strings"
        "time"
)

// ── colours ──────────────────────────────────────────────────────────────────

const (
        Reset   = "\033[0m"
        Bold    = "\033[1m"
        Dim     = "\033[2m"
        Grey    = "\033[90m"
        Red     = "\033[91m"
        Green   = "\033[92m"
        Yellow  = "\033[93m"
        Blue    = "\033[94m"
        Magenta = "\033[95m"
        Cyan    = "\033[96m"
        White   = "\033[97m"
)

func oinfo(msg string)    { fmt.Printf("%s  [%s*%s%s]%s %s\n", Cyan, Bold, Reset, Cyan, Reset, msg) }
func osuccess(msg string) { fmt.Printf("%s  [%s+%s%s]%s %s%s%s\n", Green, Bold, Reset, Green, Reset, Green, msg, Reset) }
func owarn(msg string)    { fmt.Printf("%s  [%s!%s%s]%s %s%s%s\n", Yellow, Bold, Reset, Yellow, Reset, Yellow, msg, Reset) }
func oerror(msg string)   { fmt.Printf("%s  [%s-%s%s]%s %s%s%s\n", Red, Bold, Reset, Red, Reset, Red, msg, Reset) }
func ostep(msg string)    { fmt.Printf("%s  [%s>%s%s]%s %s%s%s%s\n", Magenta, Bold, Reset, Magenta, Reset, White, Bold, msg, Reset) }
func odetail(msg string)  { fmt.Printf("%s      %s%s\n", Grey, msg, Reset) }
func odivider()           { fmt.Printf("%s  %s%s\n", Grey, strings.Repeat("-", 60), Reset) }

func oheader(title string) {
        fmt.Println()
        odivider()
        fmt.Printf("%s  %s%s%s\n", White, Bold, title, Reset)
        odivider()
        fmt.Println()
}

func printOVPNBanner() {
        fmt.Printf(`
%s  +============================================================+%s
%s  |%s  %s%s  ___  __   __ ____  _   _  %s                        %s|%s
%s  |%s  %s%s / _ \ \ \ / /|  _ \| \ | | %s                        %s|%s
%s  |%s  %s%s| | | | \ V / | |_) |  \| | %s                        %s|%s
%s  |%s  %s%s| |_| |  \_/  |  __/| |\  | %s                        %s|%s
%s  |%s  %s%s \___/        |_|   |_| \_| %s                        %s|%s
%s  |%s                                                            %s|%s
%s  |%s  %sVPNGate OVPN Config Fetcher%s  %sgithub.com/krainium%s   %s|%s
%s  +============================================================+%s
`,
                Cyan, Reset,
                Cyan, Reset, Yellow, Bold, Reset, Cyan, Reset,
                Cyan, Reset, Yellow, Bold, Reset, Cyan, Reset,
                Cyan, Reset, Yellow, Bold, Reset, Cyan, Reset,
                Cyan, Reset, Yellow, Bold, Reset, Cyan, Reset,
                Cyan, Reset, Yellow, Bold, Reset, Cyan, Reset,
                Cyan, Reset, Cyan, Reset,
                Cyan, Reset, White, Reset, Grey, Reset, Cyan, Reset,
                Cyan, Reset,
        )
        odivider()
        fmt.Println()
}

// ── VPN server record ─────────────────────────────────────────────────────────

type Server struct {
        HostName     string
        IP           string
        Score        int64
        Ping         int64
        Speed        int64
        CountryLong  string
        CountryShort string
        Sessions     int64
        LogType      string
        Operator     string
        OVPNBase64   string
}

func (s Server) SpeedMbps() float64 { return float64(s.Speed) / 1_000_000 }

func (s Server) ScoreLabel() string {
        if s.Score > 500000 {
                return Green + "★★★" + Reset
        }
        if s.Score > 100000 {
                return Yellow + "★★☆" + Reset
        }
        return Grey + "★☆☆" + Reset
}

func (s Server) PingLabel() string {
        if s.Ping == 0 {
                return Grey + "N/A" + Reset
        }
        if s.Ping < 50 {
                return Green + fmt.Sprintf("%dms", s.Ping) + Reset
        }
        if s.Ping < 150 {
                return Yellow + fmt.Sprintf("%dms", s.Ping) + Reset
        }
        return Red + fmt.Sprintf("%dms", s.Ping) + Reset
}

// ── fetch & parse ─────────────────────────────────────────────────────────────

var mirrors = []string{
        "http://www.vpngate.net/api/iphone/",
        "https://www.vpngate.net/api/iphone/",
}

func fetchCSV() ([]byte, error) {
        client := &http.Client{Timeout: 25 * time.Second}
        for _, url := range mirrors {
                resp, err := client.Get(url)
                if err != nil {
                        continue
                }
                defer resp.Body.Close()
                data, err := io.ReadAll(resp.Body)
                if err != nil || len(data) < 100 {
                        continue
                }
                return data, nil
        }
        return nil, fmt.Errorf("all mirrors failed — check your connection")
}

func parseServers(data []byte) ([]Server, error) {
        raw := string(data)
        var lines []string
        var headerLine string

        for _, line := range strings.Split(raw, "\n") {
                line = strings.TrimRight(line, "\r")
                trimmed := strings.TrimSpace(line)
                if trimmed == "" {
                        continue
                }
                if trimmed == "*" || strings.HasPrefix(trimmed, "*vpn_servers") {
                        continue
                }
                if strings.HasPrefix(trimmed, "#HostName") || strings.HasPrefix(trimmed, "#hostname") {
                        headerLine = strings.TrimLeft(trimmed, "#")
                        continue
                }
                if strings.HasPrefix(trimmed, "#") {
                        continue
                }
                lines = append(lines, line)
        }

        if headerLine == "" {
                return nil, fmt.Errorf("header row not found")
        }
        if len(lines) == 0 {
                return nil, fmt.Errorf("no server data in response")
        }

        r := csv.NewReader(strings.NewReader(headerLine + "\n" + strings.Join(lines, "\n")))
        r.FieldsPerRecord = -1
        r.LazyQuotes = true
        r.TrimLeadingSpace = true

        records, err := r.ReadAll()
        if err != nil {
                return nil, fmt.Errorf("CSV parse error: %w", err)
        }
        if len(records) < 2 {
                return nil, fmt.Errorf("no records found")
        }

        colIdx := make(map[string]int)
        for i, h := range records[0] {
                colIdx[strings.TrimLeft(h, "#")] = i
        }

        col := func(row []string, name string) string {
                idx, ok := colIdx[name]
                if !ok || idx >= len(row) {
                        return ""
                }
                return strings.TrimSpace(row[idx])
        }
        toInt := func(s string) int64 {
                v, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
                return v
        }

        var servers []Server
        for _, row := range records[1:] {
                if len(row) < 14 {
                        continue
                }
                b64 := col(row, "OpenVPN_ConfigData_Base64")
                if b64 == "" {
                        continue
                }
                servers = append(servers, Server{
                        HostName:     col(row, "HostName"),
                        IP:           col(row, "IP"),
                        Score:        toInt(col(row, "Score")),
                        Ping:         toInt(col(row, "Ping")),
                        Speed:        toInt(col(row, "Speed")),
                        CountryLong:  col(row, "CountryLong"),
                        CountryShort: strings.ToUpper(col(row, "CountryShort")),
                        Sessions:     toInt(col(row, "NumVpnSessions")),
                        LogType:      col(row, "LogType"),
                        Operator:     col(row, "Operator"),
                        OVPNBase64:   b64,
                })
        }
        return servers, nil
}

// ── display ───────────────────────────────────────────────────────────────────

func printTable(servers []Server) {
        oheader(fmt.Sprintf("VPN Servers  (%d)", len(servers)))
        fmt.Printf("  %s%-4s  %-15s  %-20s  %-18s  %-22s  %-7s  %s%s\n",
                Bold+White, "#", "IP", "Ping", "Speed", "Country", "Sess", "Score", Reset)
        odivider()
        for i, s := range servers {
                mbps := s.SpeedMbps()
                speed := ""
                switch {
                case mbps >= 100:
                        speed = Green + fmt.Sprintf("%.0fMbps", mbps) + Reset
                case mbps >= 10:
                        speed = Yellow + fmt.Sprintf("%.0fMbps", mbps) + Reset
                default:
                        speed = Red + fmt.Sprintf("%.1fMbps", mbps) + Reset
                }
                country := fmt.Sprintf("%s%s%s (%s%s%s)",
                        Cyan, s.CountryShort, Reset, Grey, truncate(s.CountryLong, 12), Reset)
                fmt.Printf("  %s%-4d%s  %-15s  %-20s  %-18s  %-22s  %-7s  %s\n",
                        Grey, i+1, Reset,
                        s.IP,
                        s.PingLabel(),
                        speed,
                        country,
                        fmt.Sprintf("%s%d%s", Magenta, s.Sessions, Reset),
                        s.ScoreLabel(),
                )
        }
        fmt.Println()
}

func truncate(s string, n int) string {
        if len(s) <= n {
                return s
        }
        return s[:n-1] + "…"
}

// ── save ──────────────────────────────────────────────────────────────────────

func saveOVPN(s Server, outDir string, idx int) (string, error) {
        decoded, err := base64.StdEncoding.DecodeString(s.OVPNBase64)
        if err != nil {
                decoded, err = base64.RawStdEncoding.DecodeString(s.OVPNBase64)
                if err != nil {
                        return "", fmt.Errorf("base64 decode: %w", err)
                }
        }
        safe := strings.Map(func(r rune) rune {
                if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
                        return r
                }
                return '_'
        }, s.HostName)
        filename := fmt.Sprintf("%03d_%s_%s.ovpn", idx, s.CountryShort, safe)
        path := filepath.Join(outDir, filename)
        if err := os.WriteFile(path, decoded, 0644); err != nil {
                return "", err
        }
        return path, nil
}

func doSave(servers []Server, outDir string) {
        if err := os.MkdirAll(outDir, 0755); err != nil {
                oerror("Cannot create directory: " + err.Error())
                return
        }
        oheader(fmt.Sprintf("Saving %d configs → %s%s%s", len(servers), Cyan, outDir, Reset))
        saved, failed := 0, 0
        start := time.Now()
        for i, s := range servers {
                path, err := saveOVPN(s, outDir, i+1)
                if err != nil {
                        oerror(fmt.Sprintf("%-15s  %s", s.IP, err.Error()))
                        failed++
                        continue
                }
                osuccess(fmt.Sprintf("%-15s  %s%s%s  %s%.0fMbps%s  → %s%s%s",
                        s.IP,
                        Cyan, s.CountryShort, Reset,
                        Green, s.SpeedMbps(), Reset,
                        Grey, filepath.Base(path), Reset,
                ))
                saved++
        }
        elapsed := time.Since(start).Round(time.Millisecond)
        fmt.Println()
        odivider()
        odetail(fmt.Sprintf("Saved    : %s%d configs%s", Green, saved, Reset))
        if failed > 0 {
                odetail(fmt.Sprintf("Failed   : %s%d%s", Red, failed, Reset))
        }
        odetail(fmt.Sprintf("Location : %s%s%s", Cyan, outDir, Reset))
        odetail(fmt.Sprintf("Time     : %s%s%s", Yellow, elapsed, Reset))
        fmt.Println()
        if saved > 0 {
                osuccess(fmt.Sprintf("Connect:  %sopenvpn --config %s/<file>.ovpn%s", Yellow, outDir, Reset))
        }
}

// ── sort helpers ──────────────────────────────────────────────────────────────

func sortBy(servers []Server, by string) []Server {
        out := make([]Server, len(servers))
        copy(out, servers)
        switch by {
        case "ping":
                sort.Slice(out, func(i, j int) bool {
                        if out[i].Ping == 0 {
                                return false
                        }
                        if out[j].Ping == 0 {
                                return true
                        }
                        return out[i].Ping < out[j].Ping
                })
        case "score":
                sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
        case "country":
                sort.Slice(out, func(i, j int) bool {
                        if out[i].CountryShort == out[j].CountryShort {
                                return out[i].Speed > out[j].Speed
                        }
                        return out[i].CountryShort < out[j].CountryShort
                })
        default:
                sort.Slice(out, func(i, j int) bool { return out[i].Speed > out[j].Speed })
        }
        return out
}

func filterCountry(servers []Server, codes string) []Server {
        set := map[string]bool{}
        for _, c := range strings.Split(strings.ToUpper(codes), ",") {
                set[strings.TrimSpace(c)] = true
        }
        var out []Server
        for _, s := range servers {
                if set[s.CountryShort] {
                        out = append(out, s)
                }
        }
        return out
}

func topN(servers []Server, n int) []Server {
        if n <= 0 || n >= len(servers) {
                return servers
        }
        return servers[:n]
}

// ── input helper ─────────────────────────────────────────────────────────────

var stdin = bufio.NewReader(os.Stdin)

func prompt(msg string) string {
        fmt.Printf("%s  %s›%s %s%s: %s", Cyan, Bold, Reset, White, msg, Reset)
        line, _ := stdin.ReadString('\n')
        return strings.TrimSpace(line)
}

func promptInt(msg string, fallback int) int {
        raw := prompt(msg)
        if raw == "" {
                return fallback
        }
        n, err := strconv.Atoi(raw)
        if err != nil || n <= 0 {
                owarn(fmt.Sprintf("Invalid number — using %d", fallback))
                return fallback
        }
        return n
}

// ── menu ──────────────────────────────────────────────────────────────────────

func printMenu(total int) {
        fmt.Println()
        odivider()
        fmt.Printf("  %s%sMain Menu%s  %s(%d servers loaded)%s\n", Bold, White, Reset, Grey, total, Reset)
        odivider()
        fmt.Printf("  %s[1]%s  List all servers  %s(sorted by speed)%s\n", Yellow, Reset, Grey, Reset)
        fmt.Printf("  %s[2]%s  Filter by country\n", Yellow, Reset)
        fmt.Printf("  %s[3]%s  Top N fastest servers\n", Yellow, Reset)
        fmt.Printf("  %s[4]%s  Top N lowest ping servers\n", Yellow, Reset)
        fmt.Printf("  %s[5]%s  Top N by score\n", Yellow, Reset)
        fmt.Printf("  %s[6]%s  Save all configs  %s(all %d servers)%s\n", Yellow, Reset, Grey, total, Reset)
        fmt.Printf("  %s[7]%s  Save by country\n", Yellow, Reset)
        fmt.Printf("  %s[8]%s  Save top N fastest\n", Yellow, Reset)
        fmt.Printf("  %s[9]%s  Refresh server list\n", Yellow, Reset)
        fmt.Printf("  %s[0]%s  %sQuit%s\n", Red, Reset, Red, Reset)
        odivider()
        fmt.Println()
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
        printOVPNBanner()

        ostep("Loading VPNGate server list ...")
        fmt.Println()

        data, err := fetchCSV()
        if err != nil {
                oerror("Fetch failed: " + err.Error())
                os.Exit(1)
        }

        servers, err := parseServers(data)
        if err != nil {
                oerror("Parse failed: " + err.Error())
                os.Exit(1)
        }

        servers = sortBy(servers, "speed")
        osuccess(fmt.Sprintf("Ready — %s%d%s servers loaded from VPNGate", White+Bold, len(servers), Reset))

        for {
                printMenu(len(servers))
                choice := prompt("Select option")

                switch choice {

                case "1":
                        // list all, sorted by speed
                        list := sortBy(servers, "speed")
                        printTable(list)

                case "2":
                        // filter by country
                        code := prompt("Country code(s)  e.g. JP or US,CA,DE")
                        if code == "" {
                                owarn("No country entered")
                                continue
                        }
                        list := filterCountry(sortBy(servers, "speed"), code)
                        if len(list) == 0 {
                                owarn(fmt.Sprintf("No servers found for: %s", strings.ToUpper(code)))
                        } else {
                                printTable(list)
                        }

                case "3":
                        // top N fastest
                        n := promptInt("How many servers", 10)
                        list := topN(sortBy(servers, "speed"), n)
                        printTable(list)

                case "4":
                        // top N lowest ping
                        n := promptInt("How many servers", 10)
                        list := topN(sortBy(servers, "ping"), n)
                        printTable(list)

                case "5":
                        // top N by score
                        n := promptInt("How many servers", 10)
                        list := topN(sortBy(servers, "score"), n)
                        printTable(list)

                case "6":
                        // save all
                        outDir := prompt("Output directory  [ovpn_configs]")
                        if outDir == "" {
                                outDir = "ovpn_configs"
                        }
                        list := sortBy(servers, "speed")
                        doSave(list, outDir)

                case "7":
                        // save by country
                        code := prompt("Country code(s)  e.g. JP or US,CA")
                        if code == "" {
                                owarn("No country entered")
                                continue
                        }
                        list := filterCountry(sortBy(servers, "speed"), code)
                        if len(list) == 0 {
                                owarn(fmt.Sprintf("No servers found for: %s", strings.ToUpper(code)))
                                continue
                        }
                        outDir := prompt("Output directory  [ovpn_configs]")
                        if outDir == "" {
                                outDir = "ovpn_configs"
                        }
                        doSave(list, outDir)

                case "8":
                        // save top N fastest
                        n := promptInt("How many servers", 10)
                        outDir := prompt("Output directory  [ovpn_configs]")
                        if outDir == "" {
                                outDir = "ovpn_configs"
                        }
                        list := topN(sortBy(servers, "speed"), n)
                        doSave(list, outDir)

                case "9":
                        // refresh
                        ostep("Refreshing server list ...")
                        fresh, err := fetchCSV()
                        if err != nil {
                                oerror("Refresh failed: " + err.Error())
                                continue
                        }
                        updated, err := parseServers(fresh)
                        if err != nil {
                                oerror("Parse failed: " + err.Error())
                                continue
                        }
                        servers = sortBy(updated, "speed")
                        osuccess(fmt.Sprintf("Refreshed — %s%d%s servers loaded", White+Bold, len(servers), Reset))

                case "0", "q", "Q", "quit", "exit":
                        fmt.Println()
                        osuccess("Goodbye.")
                        fmt.Println()
                        os.Exit(0)

                default:
                        owarn(fmt.Sprintf("Unknown option: %s%s%s — enter a number from the menu", White, choice, Reset))
                }
        }
}
