package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/db"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/liturgy"
)

type commonFlags struct {
	date, service, data string
	julian, reader      bool
	standalone, typikon bool
}

func (c *commonFlags) register(fs *flag.FlagSet) {
	fs.StringVar(&c.date, "date", time.Now().Format("2006-01-02"), "date YYYY-MM-DD (civil unless -julian)")
	fs.BoolVar(&c.julian, "julian", false, "-date is a Julian (old style) date")
	fs.StringVar(&c.service, "service", "ninth_hour", "service: first_hour, third_hour, sixth_hour, ninth_hour")
	fs.StringVar(&c.data, "data", "data", "data directory (templates, rules, i18n)")
	fs.BoolVar(&c.reader, "reader", false, "reader's order, without a priest")
	fs.BoolVar(&c.standalone, "standalone", false, "serve the hour on its own: usual beginning and dismissal")
	fs.BoolVar(&c.typikon, "typikon", false, "follow the Typikon where parish practice differs")
}

func (c *commonFlags) params() liturgy.Params {
	return liturgy.Params{Priest: !c.reader, Standalone: c.standalone, Typikon: c.typikon}
}

func parseDate(s string, julian bool) (calendar.JDate, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return calendar.JDate{}, fmt.Errorf("bad date %q: %w", s, err)
	}
	if julian {
		return calendar.NewJDate(t.Year(), t.Month(), t.Day())
	}
	return calendar.FromCivil(t.Year(), t.Month(), t.Day())
}

func newGenerator(ctx context.Context, dataDir string) (*liturgy.Generator, func(), error) {
	pool, err := db.Open(ctx)
	if err != nil {
		return nil, nil, err
	}
	rules, err := liturgy.LoadRules(dataDir)
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	feasts, err := liturgy.LoadFeasts(dataDir)
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	st := &corpus.Store{Pool: pool}
	return &liturgy.Generator{
		DataDir: dataDir,
		Planner: &liturgy.Planner{Rules: rules, Feasts: feasts, Menaion: st},
		Texts:   st,
	}, pool.Close, nil
}

func cmdPlan(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("plan", flag.ExitOnError)
	var c commonFlags
	c.register(fs)
	fs.Parse(args)
	d, err := parseDate(c.date, c.julian)
	if err != nil {
		return err
	}
	g, closeFn, err := newGenerator(ctx, c.data)
	if err != nil {
		return err
	}
	defer closeFn()
	plan, err := g.Planner.Plan(ctx, d, c.service, c.params())
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(plan)
}

func cmdRender(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	var c commonFlags
	c.register(fs)
	mode := fs.String("mode", "laity", "display mode: laity | kliros")
	lang := fs.String("lang", "cu", "language")
	out := fs.String("o", "", "output file (default stdout)")
	fs.Parse(args)
	d, err := parseDate(c.date, c.julian)
	if err != nil {
		return err
	}
	g, closeFn, err := newGenerator(ctx, c.data)
	if err != nil {
		return err
	}
	defer closeFn()
	var buf bytes.Buffer
	_, err = g.Generate(ctx, liturgy.Request{
		Date: d, Service: c.service, Lang: *lang, Mode: *mode,
		Params: c.params(),
	}, &buf)
	if err != nil {
		return err
	}
	if *out == "" {
		_, err = os.Stdout.Write(buf.Bytes())
		return err
	}
	return os.WriteFile(*out, buf.Bytes(), 0o644)
}

func cmdServe(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:8080", "listen address")
	data := fs.String("data", "data", "data directory")
	fs.Parse(args)
	g, closeFn, err := newGenerator(ctx, *data)
	if err != nil {
		return err
	}
	defer closeFn()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		date := q.Get("date")
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		julian, _ := strconv.ParseBool(q.Get("julian"))
		d, err := parseDate(date, julian)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		service := q.Get("service")
		if service == "" {
			service = "ninth_hour"
		}
		lang := q.Get("lang")
		if lang == "" {
			lang = "cu"
		}
		reader, _ := strconv.ParseBool(q.Get("reader"))
		standalone, _ := strconv.ParseBool(q.Get("standalone"))
		typikon, _ := strconv.ParseBool(q.Get("typikon"))
		var buf bytes.Buffer
		_, err = g.Generate(r.Context(), liturgy.Request{
			Date: d, Service: service, Lang: lang, Mode: q.Get("mode"),
			Params: liturgy.Params{Priest: !reader, Standalone: standalone, Typikon: typikon},
		}, &buf)
		if err != nil {
			status := http.StatusInternalServerError
			var ue *liturgy.UnsupportedError
			if errors.As(err, &ue) {
				status = http.StatusNotImplemented
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(status)
			fmt.Fprintf(w, "<!doctype html><meta charset=utf-8><title>Ошибка</title><p>%s</p>", html.EscapeString(err.Error()))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(buf.Bytes())
	})
	srv := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()
	fmt.Printf("listening on http://%s/?date=YYYY-MM-DD&service=first_hour|third_hour|sixth_hour|ninth_hour&mode=laity|kliros\n", *addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
