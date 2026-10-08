"""Text-detector comparison for the on-screen text spike (spike-onscreen-text-vision.md).

Runs DBNet-family detectors (PP-OCRv3 mobile, PP-OCRv5 mobile/server) on a
folder of frames, draws the boxes, prints box counts and time per frame.

  pip install rapidocr_onnxruntime pillow numpy
  # PP-OCRv5 models: huggingface.co/PaddlePaddle/PP-OCRv5_{mobile,server}_det_onnx
  ONS=<dir with models/v5m, models/v5s> python detcmp.py FRAMES_DIR - v3m,v5m,v5s
"""
import sys, glob, os, time
import numpy as np
from PIL import Image, ImageDraw
import rapidocr_onnxruntime as r
from rapidocr_onnxruntime.ch_ppocr_v3_det.text_detect import TextDetector
P=os.path.dirname(r.__file__); O=os.environ.get("ONS", "")
MODELS={
 "v3m": (os.path.join(P,"models/ch_PP-OCRv3_det_infer.onnx"), 736, "min"),
 "v5m": (O+"/models/v5m/inference.onnx", 960, "max"),
 "v5s": (O+"/models/v5s/inference.onnx", 960, "max"),
}
def make(name):
    path, side, typ = MODELS[name]
    return TextDetector({"model_path":path,"use_cuda":False,"limit_side_len":side,"limit_type":typ,"thresh":0.3,"box_thresh":0.6,"unclip_ratio":1.5,"use_dilation":False,"score_mode":"fast"})
if __name__=='__main__':
    names=sys.argv[3].split(",")
    dets={n:make(n) for n in names}
    for f in sorted(glob.glob(sys.argv[1]+"/*.png")):
        im=Image.open(f).convert("RGB"); a=np.array(im)[:,:,::-1].copy()
        row=[os.path.basename(f)]
        for n,d in dets.items():
            t=time.time(); boxes,_=d(a); dt=time.time()-t
            row.append(f"{n}:{len(boxes)}({dt*1000:.0f}ms)")
            im2=im.copy(); dr=ImageDraw.Draw(im2)
            for b in boxes: dr.polygon([tuple(p) for p in b], outline=(255,0,0), width=3)
            im2.save(f.replace(".png",f".{n}.jpg"))
        print("  ".join(row))
