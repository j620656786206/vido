import base64, json, os, sys, urllib.request, time, glob
key=[l.split('=',1)[1].strip().strip('"') for l in open(os.path.expanduser('~/.config/vido-eval/keys.env')) if l.startswith('CLAUDE_API_KEY=')][0]
model=sys.argv[1]; rounds=int(sys.argv[2]); out=sys.argv[3]
PROMPT=("This image stacks 6 strips. Each strip is the bottom third of one frame sampled from the same video episode. "
"Decide whether the video has Chinese subtitles burned into the picture (open captions: lines of Chinese dialogue text, usually centred near the bottom). "
"Ignore channel logos, watermarks, on-screen signs, credits rolls and English-only captions. "
"Copy every Chinese subtitle line you can read exactly as written, character by character, without converting between simplified and traditional. "
"Reply with exactly one line of JSON and nothing else: {\"burned_chinese\": true|false, \"lines\": [\"...\"]}.")
res={}
usage={'in':0,'out':0}
for p in sorted(glob.glob(os.path.join(sys.argv[4],'s*.png'))):
    sid=os.path.basename(p)[1:-4]
    img=base64.b64encode(open(p,'rb').read()).decode()
    for r in range(rounds):
        body={"model":model,"max_tokens":400,"thinking":{"type":"disabled"},"messages":[{"role":"user","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":img}},{"type":"text","text":PROMPT}]}]}
        req=urllib.request.Request("https://api.anthropic.com/v1/messages",data=json.dumps(body).encode(),headers={"x-api-key":key,"anthropic-version":"2023-06-01","content-type":"application/json"})
        for attempt in range(3):
            try:
                d=json.load(urllib.request.urlopen(req,timeout=120)); break
            except Exception as e:
                err=e; time.sleep(3)
        else:
            res.setdefault(sid,[]).append({"error":str(err)}); continue
        usage['in']+=d['usage']['input_tokens']; usage['out']+=d['usage']['output_tokens']
        txt=''.join(c.get('text','') for c in d['content'])
        try:
            j=json.loads(txt[txt.index('{'):txt.rindex('}')+1])
        except Exception:
            j={"raw":txt}
        res.setdefault(sid,[]).append(j)
    print(sid, res[sid], flush=True)
json.dump({"results":res,"usage":usage},open(out,'w'),ensure_ascii=False,indent=1)
print("usage",usage)
