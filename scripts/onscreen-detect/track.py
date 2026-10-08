"""Detect text boxes on every sampled frame, link them into tracks (YORO-style).

Frames: ffmpeg -i VIDEO -vf "fps=1/2,scale=1280:-2" frames/f%05d.jpg
Run:    ONS=<models dir> python -c "import track; track.run('frames', 'v5s', 'tracks.json')"
A track = the same box (IoU > 0.3) across consecutive frames, one missed frame
allowed; each track keeps the frame where its box is largest (the one to read).
"""
import sys, glob, os, time, json
import numpy as np
from PIL import Image
sys.path.insert(0, os.path.dirname(__file__))
from detcmp import make  # reuse detector configs
STEP = 2  # seconds per frame
def rect(b):
    xs=[p[0] for p in b]; ys=[p[1] for p in b]; return (min(xs),min(ys),max(xs),max(ys))
def iou(a,b):
    x1,y1=max(a[0],b[0]),max(a[1],b[1]); x2,y2=min(a[2],b[2]),min(a[3],b[3])
    inter=max(0,x2-x1)*max(0,y2-y1); ua=(a[2]-a[0])*(a[3]-a[1])+(b[2]-b[0])*(b[3]-b[1])-inter
    return inter/ua if ua>0 else 0
def run(frames_dir, model, out):
    d=make(model); frames=sorted(glob.glob(frames_dir+"/f*.jpg"))
    per=[]; t0=time.time()
    for f in frames:
        a=np.array(Image.open(f).convert("RGB"))[:,:,::-1].copy()
        boxes,_=d(a); per.append([rect(b) for b in boxes])
    dt=time.time()-t0
    tracks=[]  # each: dict(start,end,boxes)
    active=[]
    for i,bs in enumerate(per):
        used=set(); nxt=[]
        for tr in active:
            best=None
            for j,b in enumerate(bs):
                if j in used: continue
                v=iou(tr["last"],b)
                if v>0.3 and (best is None or v>best[0]): best=(v,j)
            if best:
                used.add(best[1]); tr["last"]=bs[best[1]]; tr["end"]=i; tr["n"]+=1
                area=(tr["last"][2]-tr["last"][0])*(tr["last"][3]-tr["last"][1])
                if area>tr["best_area"]: tr["best_area"]=area; tr["best"]=i; tr["best_box"]=tr["last"]
                nxt.append(tr)
            elif i-tr["end"]<=1: nxt.append(tr)   # allow one missed frame
            else: tracks.append(tr)
        for j,b in enumerate(bs):
            if j not in used:
                a=(b[2]-b[0])*(b[3]-b[1])
                nxt.append({"start":i,"end":i,"n":1,"last":b,"best":i,"best_area":a,"best_box":b})
        active=nxt
    tracks+=active
    res={"model":model,"frames":len(frames),"sec_per_frame":round(dt/len(frames),3),
         "boxes":sum(len(b) for b in per),
         "tracks":len(tracks),"tracks_2plus":sum(1 for t in tracks if t["n"]>=2),
         "track_list":[{"start":t["start"]*STEP,"end":t["end"]*STEP,"n":t["n"],"best":t["best"]*STEP,"box":[int(x) for x in t["best_box"]]} for t in tracks]}
    json.dump(res,open(out,"w"))
    print(json.dumps({k:v for k,v in res.items() if k!="track_list"}))
if __name__=="__main__":
    run(sys.argv[1], sys.argv[2], sys.argv[3])
