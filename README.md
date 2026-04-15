# OVPN — VPNGate Config Fetcher

Pulls the live VPNGate server list, lets you browse it by speed, ping, country or score, then saves the `.ovpn` files you actually want. No account needed. No API key. Just Go.

---

## What it does

VPNGate publishes hundreds of free volunteer-run OpenVPN servers. The problem is the only way to get configs is through a clunky web page. This tool hits the API directly, parses the full list in real time, colour-codes every server by speed and ping, then drops you into a menu where you pick exactly what to save.

Everything happens from a single interactive session. No flags to memorise. No config files to edit.

---

## Requirements

- Go 1.21 or newer
- An internet connection (fetches live data from VPNGate on launch)
- OpenVPN installed if you plan to actually connect

---

## Build

```bash
git clone https://github.com/krainium/<repo>
cd A/ovpn
go build -o ovpn .
```

On Windows:

```bash
go build -o ovpn.exe .
```

---

## Run

```bash
./ovpn
```

The tool fetches the server list silently on startup then shows the menu:

```
  Main Menu  (98 servers loaded)
  ------------------------------------------------------------
  [1]  List all servers  (sorted by speed)
  [2]  Filter by country
  [3]  Top N fastest servers
  [4]  Top N lowest ping servers
  [5]  Top N by score
  [6]  Save all configs
  [7]  Save by country
  [8]  Save top N fastest
  [9]  Refresh server list
  [0]  Quit
```

Pick a number. If an option needs more input (country code, how many servers, output folder) it asks inline. Results print immediately. You go straight back to the menu.

---

## Menu options

**[1] List all servers**
Full table of every server sorted by speed. Shows IP, ping, speed in Mbps, country, active sessions, score rating.

**[2] Filter by country**
Enter a two-letter country code. Multiple codes work too — separate them with commas. Example: `JP` or `US,CA,DE`.

**[3] Top N fastest**
Enter a number. Shows that many servers from the top of the speed ranking.

**[4] Top N lowest ping**
Same as above but sorted by ping. Servers with no ping data get pushed to the bottom.

**[5] Top N by score**
VPNGate assigns a score based on uptime, speed, sessions. This surfaces the most reliable servers.

**[6] Save all configs**
Saves every server on the list as a `.ovpn` file. You choose the output folder (defaults to `ovpn_configs`).

**[7] Save by country**
Same save flow but only for the countries you specify.

**[8] Save top N fastest**
Saves a specific number of the fastest servers. Useful if you just want a shortlist to test.

**[9] Refresh**
Re-fetches the live server list without restarting the tool. VPNGate updates its list continuously so this gives you fresh data.

**[0] Quit**

---

## Connecting

Once you have a `.ovpn` file:

```bash
sudo openvpn --config ovpn_configs/001_JP_somehostname.ovpn
```

On Windows, import the file into the OpenVPN GUI.

---

## Output files

Files are named in the format:

```
001_JP_hostname.ovpn
002_CA_hostname.ovpn
```

The number prefix keeps them ordered by rank within whatever sort you used (speed, ping, score).

---

## Notes

- VPNGate servers are volunteer-run. Quality varies. Use ping, speed, score together to pick a good one.
- Logs are kept by most servers. This is a privacy tool for bypassing geo-restrictions, not for anonymity.
- If the fetch fails, the tool tries a fallback mirror automatically before giving up.
