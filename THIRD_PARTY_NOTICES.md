# Third-party notices

## Bundled in the executable

| Component | Version | License | Notes |
|---|---|---|---|
| [Leaflet](https://leafletjs.com/) | 1.9.4 | BSD-2-Clause | Vendored in `web/vendor/leaflet/` (license text in `web/vendor/leaflet/LICENSE`) |
| [wailsapp/go-webview2](https://github.com/wailsapp/go-webview2) | v1.0.23 | MIT | WebView2 bindings + pure-Go WebView2 loader |
| [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys) | v0.48.0 | BSD-3-Clause | Windows API / registry |
| [jchv/go-winloader](https://github.com/jchv/go-winloader) | (indirect) | ISC | Transitive dependency of go-webview2; only used with the `native_webview2loader` build tag (not used by default builds, not linked into the exe) |

## Build tools (not shipped)

| Tool | License |
|---|---|
| [tc-hib/go-winres](https://github.com/tc-hib/go-winres) v0.3.3 | 0BSD / MIT-style (see project) |

## Data and map services (fetched at runtime, not bundled)

- **Air quality data**: National Environment Agency (NEA), via [data.gov.sg](https://data.gov.sg/).
  Contains information from data.gov.sg accessed on the date of use, made available under the
  terms of the [Singapore Open Data Licence version 1.0](https://data.gov.sg/open-data-licence).
- **Base map**: [OneMap](https://www.onemap.gov.sg/) "Night" tiles, (c) Singapore Land Authority.
  Subject to the OneMap terms of use; attribution is shown on the map.
- **Fallback base map**: Esri World Dark Gray Canvas (Esri, HERE, Garmin, (c) OpenStreetMap contributors),
  subject to Esri's terms of use. Only used if OneMap tiles fail to load.
