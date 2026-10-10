#!/usr/bin/env python3
"""Blind-A/B: Referenz-Piper vs. go-piper inproc.

Rendert 10 deutsche Texte mit beiden Engines, lautheits-normalisiert (RMS),
und erzeugt eine blinde A/B-Hörprobe (A/B/Gleichstand).

Aufruf: abtest.py [--inproc BIN]
"""
import base64
import os
import random
import subprocess
import sys
import wave

import numpy as np

ROOT = "/workspace"
PIPER = f"{ROOT}/.tmp/wasky-piper/piper/piper"
MODEL = f"{ROOT}/.tmp/wasky-piper/orig/de_DE-thorsten-high.onnx"
CFG = f"{ROOT}/.tmp/wasky-piper/orig/de_DE-thorsten-high.onnx.json"
ESPEAK = f"{ROOT}/.tmp/wasky-piper/piper/espeak-ng-data"
OUT = f"{ROOT}/9999-go-piper/demo/ab"

INPROC = "/tmp/vits5"
if "--inproc" in sys.argv:
    INPROC = sys.argv[sys.argv.index("--inproc") + 1]

TEXTS = [
    ("Umlaute", "Schöne Grüße aus München. Die Bäume blühen, und über den Hügeln singen die Vögel."),
    ("Kurz/schnell", "Kurz und knapp."),
    ("Langsam", "Langsam und deutlich spreche ich diesen längeren Satz, damit man jeden Laut klar hören kann."),
    ("Neutral", "Ich bin ein freundlicher Assistent und lese dir deine Antworten vor."),
    ("Rauschen/Klang", "Zwischen den Wörtern liegt ein leises, gleichmäßiges Rauschen im Hintergrund."),
    ("Neutral kurz", "Guten Tag. Wie geht es dir heute?"),
    ("Pangramm", "Der schnelle braune Fuchs springt über den faulen Hund."),
    ("Umlaute 2", "Über grünen Hügeln schweben weiße Wolken und ein kalter Wind weht."),
    ("Zahlen", "Eins, zwei, drei, vier, fünf, sechs, sieben, acht, neun, zehn."),
    ("Neutral lang", "Am Dienstag treffen wir uns um vier Uhr am Bahnhof und fahren dann weiter."),
]


def render_piper(text, out):
    env = dict(os.environ)
    env["LD_LIBRARY_PATH"] = os.path.dirname(PIPER)
    subprocess.run([PIPER, "-q", "-m", MODEL, "--espeak_data", ESPEAK, "-f", out],
                   input=text.encode(), env=env, check=True,
                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


def render_inproc(text, out):
    subprocess.run([INPROC, MODEL, CFG, out, text], check=True,
                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


def read_wav(path):
    w = wave.open(path, "rb")
    sr, n = w.getframerate(), w.getnframes()
    data = np.frombuffer(w.readframes(n), dtype=np.int16).astype(np.float32)
    w.close()
    return sr, data


def write_wav(path, sr, data):
    w = wave.open(path, "wb")
    w.setnchannels(1); w.setsampwidth(2); w.setframerate(sr)
    w.writeframes(data.astype(np.int16).tobytes())
    w.close()


def rms_normalize(sr, data, target_db=-20.0):
    rms = np.sqrt(np.mean(data ** 2)) + 1e-9
    target = 10 ** (target_db / 20.0) * 32767.0
    data = data * (target / rms)
    peak = np.max(np.abs(data)) + 1e-9
    if peak > 32767:
        data = data * (32767.0 / peak) * 0.95
    return sr, data


def main():
    os.makedirs(OUT, exist_ok=True)
    rows = []
    for i, (aspect, text) in enumerate(TEXTS, 1):
        ref_raw, our_raw = f"/tmp/ab_ref_{i}.wav", f"/tmp/ab_our_{i}.wav"
        render_piper(text, ref_raw)
        render_inproc(text, our_raw)
        sr, ref = rms_normalize(*read_wav(ref_raw))
        _, our = rms_normalize(*read_wav(our_raw))
        ref_f, our_f = f"ref_{i:02d}.wav", f"our_{i:02d}.wav"
        write_wav(os.path.join(OUT, ref_f), sr, ref)
        write_wav(os.path.join(OUT, our_f), sr, our)
        flip = random.random() < 0.5
        a, b = (("P", ref_f), ("I", our_f)) if not flip else (("I", our_f), ("P", ref_f))
        rows.append({"i": i, "aspect": aspect, "text": text, "a": a, "b": b})
        print(f"  {i:02d} [{aspect}] ref={len(ref)/sr:.1f}s inproc={len(our)/sr:.1f}s")

    # ab.html wird NICHT hier erzeugt (sonst Überschreiben der Turnier-Seite).
    print("Audio fertig. ab.html bleibt unverändert (Turnier-Flow).")


def write_html(rows):
    data = []
    for r in rows:
        data.append({
            "i": r["i"], "aspect": r["aspect"], "text": r["text"],
            "a": r["a"][0], "af": r["a"][1], "b": r["b"][0], "bf": r["b"][1],
        })
    html = HTML.replace("__DATA__", "const DATA = " + repr(data) + ";")
    with open(f"{ROOT}/9999-go-piper/demo/ab.html", "w") as f:
        f.write(html)


HTML = r"""<!doctype html>
<html lang=de><head><meta charset=utf-8><title>Blind-A/B — Piper vs go-piper inproc</title>
<style>
:root{color-scheme:dark}body{margin:0;padding:34px 40px 80px;background:#0b0b0f;color:#ececf2;
font:15px/1.55 -apple-system,"SF Pro Text",system-ui,sans-serif;max-width:980px}
h1{font-size:22px;margin:0 0 4px}p.sub{color:#9a9aa5;margin:0 0 18px}
.ctrl{position:sticky;top:0;background:#0b0b0f;padding:10px 0;border-bottom:1px solid #1c1c24;margin-bottom:16px}
button{background:#3a6ea5;color:#fff;border:none;border-radius:9px;padding:9px 16px;font:inherit;font-weight:600;cursor:pointer}
button.b{background:#22324a;font-weight:400}
.sentence{border:1px solid #2a2a36;background:#161620;border-radius:12px;padding:10px 14px;margin:14px 0 8px}
.sentence .lbl{color:#8a8a96;font-size:12px;text-transform:uppercase;letter-spacing:.05em}
.row{display:grid;grid-template-columns:120px 1fr;gap:14px;align-items:center;padding:10px 14px;
border:1px solid #23232c;border-radius:12px;margin-bottom:8px;background:#14141a}
.row .pname{font-weight:600}.masked{color:#9aa0b4;font-style:italic}
audio{width:100%}.choices{display:flex;gap:10px;margin:6px 0 20px}
.choices button{flex:1;background:#191922;border:1px solid #2b2b38;color:#dcdce8;font-weight:500}
.choices button:hover{background:#23324a}
pre{white-space:pre-wrap;background:#101016;border:1px solid #23232c;border-radius:12px;padding:16px;max-width:980px}
</style></head><body>
<h1>Blind-A/B — Piper (Referenz) vs. go-piper inproc</h1>
<p class=p.sub>Pro Text zwei Proben (A/B, zufällig). Wähle <b>A besser</b>, <b>B besser</b> oder <b>Gleichstand</b>.</p>
<div class=ctrl><button id=go>Start</button> <button class=b id=reveal>alle aufdecken</button> <span id=status class=sub></span></div>
<div id=app></div>
<script>
__DATA__
const $=(id)=>document.getElementById(id);const app=$("app");let idx=0;
const letters=["A","B"];
function el(h){const t=document.createElement("template");t.innerHTML=h.trim();return t.content.firstElementChild;}
function render(){
  app.innerHTML="";
  DATA.forEach((d,di)=>{
    const sec=el(`<section><div class="sentence"><div class="lbl">${d.i} · ${d.aspect}</div>${d.text}</div>
      <div class="row"><div class="pname masked">Probe A</div><audio controls preload=none src="ab/${d.af}"></audio></div>
      <div class="row"><div class="pname masked">Probe B</div><audio controls preload=none src="ab/${d.bf}"></audio></div>
      <div class="choices"><button data-p="A">A besser</button><button data-p="tie">Gleichstand</button><button data-p="B">B besser</button></div></section>`);
    sec.querySelectorAll(".choices button").forEach(b=>b.onclick=()=>{
      DATA[di].choice=b.dataset.p;
      sec.querySelectorAll(".choices button").forEach(x=>x.style.background="#191922");
      b.style.background="#2c4a6e";
      updateStatus();
    });
    app.appendChild(sec);
  });
}
function updateStatus(){const done=DATA.filter(d=>d.choice).length;$("status").textContent=done+" / "+DATA.length+" beantwortet";}
$("go").onclick=render;
$("reveal").onclick=()=>{
  document.querySelectorAll(".pname").forEach((n,i)=>{
    const d=DATA[Math.floor(i/2)];
    const isA=(i%2===0);
    const who=isA?d.a:d.b;
    n.classList.remove("masked");n.textContent=(isA?"Probe A":"Probe B")+" — "+(who==="P"?"Piper (Ref)":"go-piper inproc");
  });
};
render();
</script></body></html>
"""

if __name__ == "__main__":
    main()
