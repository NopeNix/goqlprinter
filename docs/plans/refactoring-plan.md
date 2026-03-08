# Refaktorointisuunnitelma: goqlprinter

> **KRIITTINEN OHJE AI-AGENTILLE:**
> Tämä on **refaktorointiprojekti**. Nykyinen koodipohja sisältää anti-patterneja ja rakenteellisia virheitä.
> **ÄLÄ ota mallia olemassa olevasta koodista.** Toteuta jokainen vaihe Go:n parhaiden käytäntöjen mukaan,
> vaikka se poikkeaisi täysin nykyisestä toteutuksesta. Olemassa oleva koodi on se mitä korjataan — ei esimerkki
> jota noudattaa.
>
> Konkreettisesti:
> - Jos näet globaalin muuttujan, **älä luo uutta globaalia** — injektoi riippuvuus
> - Jos näet `log.Printf`-kutsun, **älä käytä log.Printf** — käytä `log/slog`
> - Jos näet `...interface{}`, **älä kirjoita interface{}** — käytä `any`
> - Jos näet paketin ilman testejä, **kirjoita testit ensin**
> - Jos näet virheen joka ohitetaan `_`:lla, **käsittele virhe**

---

> **PAKOLLINEN: Työkalut ja agentit**
>
> ### 1. Lataa Go-skill (`golang-pro`) AINA ennen koodaustyötä
> Jokaisen vaiheen alussa: kutsu `golang-pro` skill. Se tuo Go-spesifiset parhaat käytännöt,
> idiomit ja tarkistukset kontekstiin. **Älä aloita koodausta ilman tätä.**
>
> ### 2. Käytä LSP:tä (gopls) aktiivisesti
> - **`goToDefinition`** — ennen kuin muokkaat funktiota, tarkista mistä se on peräisin
> - **`findReferences`** — ennen kuin muutat signaturea, selvitä kaikki kutsupaikat
> - **`documentSymbol`** — hahmota tiedoston rakenne ennen muokkausta
> - **`hover`** — tarkista tyypit ja rajapinnat epäselvissä tilanteissa
> - **`goToImplementation`** — selvitä rajapintojen toteutukset ennen DI-refaktorointia
>
> LSP on nopeampi ja tarkempi kuin grep-haku symbolien jäljittämiseen. **Käytä sitä ensisijaisena
> navigointityökaluna.**
>
> ### 3. Käytä ast-grep -työkalua koodin muokkaukseen
> Järjestelmään on asennettu **ast-grep** (`sg`) — AST-pohjainen hakutyökalu joka ymmärtää
> Go:n syntaksipuun. Käytä sitä erityisesti:
> - **Mekaanisiin massamuutoksiin** (esim. `interface{}` → `any`, `log.Printf` → `slog.Info`)
> - **Koodihakuihin** jotka vaativat rakenteellista ymmärrystä (ei pelkkää tekstihakua)
> - **Anti-pattern-tunnistukseen** (globaalit muuttujat, ohitetut virheet, vanhat idiomit)
> - Käytä `ast-python-agent` refaktorointiin, `ast-code-reviewer` laaduntarkistukseen,
>   `ast-research-agent` koodipohjan analysointiin
>
> ast-grep on **huomattavasti tarkempi** kuin regex-pohjaiset haut koska se operoi syntaksipuulla.
>
> ### 4. Valitse oikea agenttimalli jokaiseen tehtävään
>
> | Tehtävätyyppi | Malli | Perustelu |
> |---------------|-------|-----------|
> | Arkkitehtuuripäätökset, DI-suunnittelu, rajapintamuutokset | **Opus** | Vaatii syvää päättelyä ja kokonaisuuden hallintaa |
> | Koodin kirjoitus, refaktorointi, testien toteutus | **Sonnet** | Nopea ja tarkka koodaustyöhön, hyvä hinta/laatu |
> | Mekaaniset massamuutokset, import-päivitykset, formatointi | **Haiku** | Nopein yksinkertaisiin, toistuviin operaatioihin |
> | Koodikatselmukset, bugianalyysi | **Opus** | Löytää hienojakoiset ongelmat ja loogiset virheet |
> | Testien generointi (table-driven, mock-setup) | **Sonnet** | Tehokas rakenteelliseen koodiin |
> | Dokumentaation päivitys, kommenttien kirjoitus | **Haiku** | Riittävä selkeisiin, rajattuihin tehtäviin |
>
> **Älä käytä Opusta mekaaniseen työhön.** Älä käytä Haikua arkkitehtuuripäätöksiin.
> Oikea malli oikeaan tehtävään säästää aikaa ja rahaa.

## Tilanne nyt

### go vet -virheet
- `services/font_service.go:139`: ei-vakio format-merkkijono `fmt.Errorf`-kutsussa

### Rakenteelliset ongelmat (vakavuusjärjestyksessä)

| # | Ongelma | Vaikutus |
|---|---------|---------|
| 1 | Nolla testiä koko projektissa | Refaktorointi ilman testejä on arpapeliä |
| 2 | Globaali muuttuva tila (`config.Cfg`, `services.defaultProvider`, `services.activeDefaultPrinter`) | Testaamaton, kilpailutilanteille altis |
| 3 | Ei `internal/`-pakettia — kaikki paketit ovat julkisia | API-pinta on hallitsemattoman laaja |
| 4 | Oma logger `log`-paketin ympärillä sen sijaan, että käyttäisi `log/slog` (Go 1.21+) | Turhaa koodia, ei strukturoitua loggausta |
| 5 | Epäyhtenäinen loggaus — `config/` käyttää `log.Printf`, muu koodi `logger.*` | Sekava lokituloste |
| 6 | Rekursiivinen kutsu `initializeBackendProvider` (main.go:82-83) | Mahdollinen stack overflow |
| 7 | Moduulinimi `goqlprinter` ei noudata `github.com/user/repo` -konventiota | Vaikeuttaa ulkoista käyttöä |
| 8 | `...interface{}` kaikkialla `...any`:n sijaan | Vanhentunut Go <1.18 -tyyli |

---

## Vaiheet

Jokainen vaihe on itsenäinen commit. Vaiheiden järjestys on kriittinen — myöhemmät vaiheet riippuvat aiemmista.

---

### Vaihe 0: Korjaa go vet -virhe

**Tiedosto:** `services/font_service.go:139`

**Ongelma:**
```go
return "", fmt.Errorf(errMsg) // ei-vakio format string — mahdollistaa format-injektion
```

**Korjaus:**
```go
return "", fmt.Errorf("font %q not found in any searched directories", fontFamily)
```

Poista turha `errMsg`-välimuuttuja kokonaan.

**Valmistuskriteeri:** `go vet ./...` palauttaa 0 virhettä.

---

### Vaihe 1: Korvaa logger `log/slog`-paketilla

> **HUOM:** Älä paranna nykyistä loggeria — poista se kokonaan ja käytä standardikirjastoa.

**Poistettava:** `logger/logger.go` (koko paketti)

**Toteutus:**
1. Luo `internal/logging/logging.go` — ohut wrapper joka alustaa `slog.Logger`-instanssin
2. Tarjoa `Init(level string) *slog.Logger` joka palauttaa konfiguroitun loggerin
3. Korvaa **kaikki** `logger.Debug/Info/Warning/Error`-kutsut `slog.Debug/Info/Warn/Error`-kutsuilla
4. Korvaa **kaikki** `log.Printf`-kutsut `config/config.go`-tiedostossa `slog`-kutsuilla
5. Poista `logger/`-paketti

**Malli:**
```go
// internal/logging/logging.go
package logging

import (
    "log/slog"
    "os"
)

func Init(level string) *slog.Logger {
    var lvl slog.Level
    switch level {
    case "DEBUG":
        lvl = slog.LevelDebug
    case "WARNING", "WARN":
        lvl = slog.LevelWarn
    case "ERROR":
        lvl = slog.LevelError
    default:
        lvl = slog.LevelInfo
    }
    h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
    return slog.New(h)
}
```

**Valmistuskriteeri:** `grep -r '"goqlprinter/logger"' .` palauttaa 0 osumaa. Kaikki loggaus kulkee `slog`:n kautta.

---

### Vaihe 2: Siirrä sisäiset paketit `internal/`-alle

**Siirrot:**
```
services/          → internal/services/
config/config.go   → internal/config/config.go
```

`brotherql/` jää juuritasolle — se on projektin ydinkirjasto ja voisi olla erillinen moduuli.
`api/` jää juuritasolle — se on julkinen HTTP-rajapinta.

**Päivitä kaikki import-polut.** Go-moduulijärjestelmä ei tee tätä automaattisesti.

**Valmistuskriteeri:** `go build ./...` onnistuu. Ulkoiset importit `services/` ja `config/`-paketteihin eivät ole enää mahdollisia.

---

### Vaihe 3: Poista globaali tila — dependency injection

> **KRIITTINEN:** Tämä on refaktoroinnin ydin. Älä kopioi nykyistä globaalia mallia.
> Nykyinen koodi käyttää globaaleja muuttujia ja pakettitason funktioita.
> Uusi koodi käyttää struct-pohjaista riippuvuusinjektiota.

**Poistettavat globaalit:**

| Muuttuja | Sijainti | Korvike |
|----------|----------|---------|
| `config.Cfg` | `config/config.go:14` | `LoadConfig()` palauttaa `*Config`, ei tallenna globaaliin |
| `services.defaultProvider` | `services/printer_service.go:23` | Struct-kenttä `PrinterService.provider` |
| `services.activeDefaultPrinter` | `services/printer_service.go:18` | Struct-kenttä `PrinterService.defaultPrinter` |
| `services.PrinterLock` | `services/printer_service.go:15` | Struct-kenttä `PrinterService.mu` |
| `globalBackendProvider` | `main.go:33` | Paikallinen muuttuja `main()`-funktiossa |

**Uusi rakenne:**

```go
// internal/config/config.go
func LoadConfig() (*Config, error) { ... } // palauttaa, ei tallenna

// internal/services/printer_service.go
type PrinterService struct {
    mu             sync.Mutex
    provider       brotherql.BackendProvider
    defaultPrinter *FoundPrinter
    log            *slog.Logger
}

func NewPrinterService(provider brotherql.BackendProvider, log *slog.Logger) *PrinterService { ... }
func (s *PrinterService) FindPrinters() ([]FoundPrinter, error) { ... }
func (s *PrinterService) ResolvePrinter(id string) (FoundPrinter, error) { ... }

// internal/services/font_service.go
type FontService struct {
    fontDirs []string
    log      *slog.Logger
}

func NewFontService(dirs []string, log *slog.Logger) *FontService { ... }
```

**API-handlerit:**

```go
// api/handlers.go
type Handlers struct {
    Printers *services.PrinterService
    Fonts    *services.FontService
    Config   *config.Config
    Log      *slog.Logger
}

func NewHandlers(ps *services.PrinterService, fs *services.FontService, cfg *config.Config, log *slog.Logger) *Handlers { ... }

// api/print.go
func (h *Handlers) PrintLabel(c *gin.Context) { ... }
```

**main.go wiring:**

```go
func main() {
    cfg, err := config.LoadConfig()
    // ...
    log := logging.Init(logLevel)
    provider := initializeBackendProvider(cfg, log)
    ps := services.NewPrinterService(provider, log)
    fs := services.NewFontService(cfg.App.FontDirs, log)
    handlers := api.NewHandlers(ps, fs, cfg, log)
    router := setupRouter(handlers, log)
    // ...
}
```

**Valmistuskriteeri:** `grep -rn 'var .* = ' --include='*.go' | grep -v 'embed\|//go:' ` ei palauta pakettitason muuttuvia globaaleja (const ja embed OK).

---

### Vaihe 4: Korjaa rekursiivinen initializeBackendProvider

**Ongelma:** `main.go:82-83` — tuntematon backend-arvo aiheuttaa rekursiivisen kutsun.

**Korjaus:** Poista rekursio, käytä fallthrough-logiikkaa:

```go
func initializeBackendProvider(cfg *config.Config, log *slog.Logger) brotherql.BackendProvider {
    backend := cfg.App.Backend
    switch backend {
    case "usb":
        return initUSBProvider()
    case "native":
        return createNativeProvider()
    case "auto":
        return autoSelectProvider(log)
    default:
        log.Warn("unknown backend, falling back to auto", "backend", backend)
        return autoSelectProvider(log)
    }
}
```

---

### Vaihe 5: Kirjoita testit

> **HUOM:** Testit kirjoitetaan **vaiheessa 3 luotujen structien** pohjalta, ei nykyisen globaalin
> tilan pohjalta. Struct-pohjainen DI mahdollistaa mock-injektoinnin.

**Prioriteettijärjestys:**

1. **`brotherql/raster_test.go`** — kuvanprosessointi on puhdasta logiikkaa, helppo testata
2. **`brotherql/models_test.go`** — mallit ja label-koot
3. **`brotherql/brotherql_test.go`** — protokollakomennot mock-backendin kanssa
4. **`internal/services/printer_service_test.go`** — mock `BackendProvider`
5. **`internal/services/font_service_test.go`** — mock-tiedostojärjestelmä
6. **`internal/config/config_test.go`** — ympäristömuuttujat ja oletusarvot
7. **`api/*_test.go`** — HTTP-handlerit `httptest`-paketilla

**Testien periaatteet:**
- Table-driven testit subtesteinä (`t.Run`)
- `-race` lippu kaikissa testeissä
- Mock-toteutukset `BackendProvider` ja `Backend` -rajapinnoille
- `t.Parallel()` kaikissa testeissä joissa mahdollista
- Ei globaalien muuttujien käyttöä testeissä — kaikki injektoidaan

**Mock-esimerkki:**
```go
type mockBackendProvider struct {
    printers []brotherql.PrinterInfo
    err      error
}

func (m *mockBackendProvider) FindPrinters() ([]brotherql.PrinterInfo, error) {
    return m.printers, m.err
}

func (m *mockBackendProvider) Connect(uri string) (brotherql.Backend, error) {
    return &mockBackend{}, nil
}

func (m *mockBackendProvider) SupportsStatus() bool { return false }
```

**Valmistuskriteeri:** `go test -race -count=1 ./...` läpäisee. Kattavuus vähintään 60% ydinpaketeissa.

---

### Vaihe 6: Päivitä moduulinimi (valinnainen)

Vaihda `go.mod`:n moduulinimi muotoon joka noudattaa konventiota:
```
module github.com/<user>/goqlprinter
```

Tämä vaatii **kaikkien** import-polkujen päivityksen. Suorita vasta kun kaikki muu on valmis ja testit läpäisevät.

---

---

## Kriittiset huomiot suunnitelman järjestykseen

> **HUOM: Lue nämä ennen toteutusta — ne korjaavat alkuperäisen suunnitelman heikkouksia.**

### Huomio 1: Vaihe 4 yhdistetään vaiheeseen 3

Rekursiivinen `initializeBackendProvider` (nykyinen vaihe 4) on osa `main.go`:n wiring-logiikkaa, joka
kirjoitetaan joka tapauksessa uusiksi vaiheessa 3 (DI-refaktorointi). **Erillinen vaihe on turha.**
Korjaa rekursio osana vaiheen 3 main.go-wiringiä.

### Huomio 2: Vaihe 2 ja 3 kannattaa yhdistää

Vaihe 2 siirtää tiedostot `internal/`-alle ja päivittää importit. Vaihe 3 kirjoittaa samat tiedostot
uusiksi (globaalit → structit). Jos nämä tehdään erikseen, **import-polut päivitetään kahdesti** —
ensin siirron yhteydessä, sitten DI-refaktoroinnissa.

**Parempi tapa:** Tee DI-refaktorointi suoraan `internal/`-rakenteeseen yhdessä vaiheessa.
Luo `internal/services/` ja `internal/config/` uudella struct-pohjaisella koodilla kerralla.

### Huomio 3: Testit pitää kirjoittaa ENNEN refaktorointia (osittain)

Suunnitelma tunnistaa ongelman ("refaktorointi ilman testejä on arpapeliä") mutta tekee juuri niin
vaiheiden 1–4 ajan. **`brotherql/`-paketti on puhdasta logiikkaa** — sille voi ja pitää kirjoittaa
testit nykyistä koodia vasten ennen kuin mitään refaktoroidaan:

- `brotherql/raster_test.go` — kuvanprosessointi, threshold-logiikka, PackBits
- `brotherql/models_test.go` — mallit, label-koot, validointi

Nämä testit toimivat **suojaverkkona** koko refaktoroinnin ajan ja varmistavat ettei ydinlogiikka rikkoudu.

### Huomio 4: slog-wrapperin ristiriita

Vaiheen 1 `logging.Init()` palauttaa `*slog.Logger`-instanssin, mutta esimerkkikoodi käyttää
`slog.Debug/Info`-pakettitason kutsuja. Valitse **yksi linja**:

- **Vaihtoehto A (suositus):** `slog.SetDefault(log)` main.go:ssa → käytä `slog.Info()` kaikkialla.
  Yksinkertaisin, ei tarvitse injektoida loggeria jokaiseen structiin.
- **Vaihtoehto B:** Injektoi `*slog.Logger` structeihin → käytä `s.log.Info()` metodeissa.
  Testattavampi, mutta lisää boilerplatea.

Älä tee molempia sekaisin.

---

## Korjattu suoritusjärjestys

```
Vaihe 0   →  go vet -korjaus + interface{} → any (ast-grep)     [Haiku]   (10 min)
Vaihe 0b  →  brotherql/-testit nykyistä koodia vasten            [Sonnet]  (45 min)
Vaihe 1   →  slog-migraatio, logger/-paketin poisto              [Sonnet]  (30 min)
Vaihe 2   →  internal/-siirto + DI-refaktorointi + rekursion     [Opus]    (2-3 h)
            poisto (entiset vaiheet 2+3+4 yhdistettynä)
Vaihe 3   →  loput testit (services, config, api)                [Sonnet]  (2 h)
Vaihe 4   →  moduulinimi (valinnainen)                            [Haiku]   (15 min)
```

Jokainen vaihe on oma committinsa. Testit ajetaan jokaisen vaiheen jälkeen: `go build ./... && go vet ./... && go test ./...`

## Ensimmäinen aalto — riskitön, heti toteutettava

Nämä korjaukset ovat itsenäisiä eivätkä aiheuta merge-konflikteja myöhempiin vaiheisiin:

| Korjaus | Työkalu | Malli | Aika |
|---------|---------|-------|------|
| `font_service.go:139` vet-korjaus | LSP → findReferences, sitten Edit | Haiku | 1 min |
| `interface{}` → `any` koko projektissa | ast-grep massamuutos | Haiku | 5 min |
| Rekursion poisto `main.go:82-83` | LSP → hover/goToDefinition | Sonnet | 5 min |
| `brotherql/raster_test.go` + `models_test.go` | golang-pro skill, testien kirjoitus | Sonnet | 45 min |

## Tarkistuslista ennen valmistumista

- [ ] `go vet ./...` palauttaa 0 virhettä
- [ ] `go build ./...` onnistuu kaikilla build-tageilla (`usb`, `!usb`)
- [ ] `go test -race ./...` läpäisee
- [ ] Ei pakettitason muuttuvia globaaleja (paitsi `embed`)
- [ ] Kaikki loggaus kulkee `slog`:n kautta
- [ ] `internal/`-paketit suojaavat sisäiset rajapinnat
- [ ] Jokainen exported-funktio on dokumentoitu
- [ ] Ei `...interface{}` — vain `...any`
- [ ] LSP:tä käytetty kaikkien symbolimuutosten verifiointiin
- [ ] ast-grep käytetty mekaanisiin massamuutoksiin (ei manuaalista sed/regex)
