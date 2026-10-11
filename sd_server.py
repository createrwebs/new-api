import os
import time
import io
import base64
import uuid
from typing import Optional, List
from fastapi import FastAPI, HTTPException, Request
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel
import torch
from diffusers import AutoPipelineForText2Image
from PIL import Image

app = FastAPI(title="Tora API Text-to-Image Service", version="1.0.0")

# Static directory for image URLs
IMAGE_DIR = "/home/ubuntu/generated_images"
os.makedirs(IMAGE_DIR, exist_ok=True)
app.mount("/images", StaticFiles(directory=IMAGE_DIR), name="images")

# Load model globally on startup
print("Loading SD-Turbo model on CPU...")
start_t = time.time()
pipe = AutoPipelineForText2Image.from_pretrained(
    "stabilityai/sd-turbo",
    torch_dtype=torch.float32
)
pipe.to("cpu")
pipe.set_progress_bar_config(disable=True)
print(f"SD-Turbo loaded successfully in {time.time() - start_t:.2f}s!")

class ImageGenerationRequest(BaseModel):
    prompt: str
    model: Optional[str] = "sd-turbo"
    n: Optional[int] = 1
    size: Optional[str] = "512x512"
    response_format: Optional[str] = "b64_json" # or "url"

class ImageDataItem(BaseModel):
    b64_json: Optional[str] = None
    url: Optional[str] = None
    revised_prompt: Optional[str] = None

class ImageGenerationResponse(BaseModel):
    created: int
    data: List[ImageDataItem]

@app.get("/health")
def health():
    return {"status": "ok", "service": "toraapi-sd-image", "model": "sd-turbo"}

@app.get("/v1/models")
def list_models():
    return {
        "object": "list",
        "data": [
            {"id": "sd-turbo", "object": "model", "owned_by": "stabilityai"},
            {"id": "dall-e-2", "object": "model", "owned_by": "openai"},
            {"id": "dall-e-3", "object": "model", "owned_by": "openai"}
        ]
    }

@app.post("/v1/images/generations", response_model=ImageGenerationResponse)
async def generate_image(req: ImageGenerationRequest, request: Request):
    if not req.prompt:
        raise HTTPException(status_code=400, detail="Prompt is required")
    
    n = min(max(req.n or 1, 1), 4) # bound 1..4
    
    # Parse width & height
    width, height = 512, 512
    if req.size:
        parts = req.size.lower().split("x")
        if len(parts) == 2 and parts[0].isdigit() and parts[1].isdigit():
            width = min(int(parts[0]), 512)
            height = min(int(parts[1]), 512)
            
    print(f"Generating {n} image(s) for prompt: '{req.prompt[:60]}...' (size: {width}x{height})")
    
    data_items = []
    host = request.headers.get("host", "13.239.38.104:8000")
    
    for i in range(n):
        t0 = time.time()
        # 1-step SD-Turbo inference
        result = pipe(
            prompt=req.prompt,
            num_inference_steps=1,
            guidance_scale=0.0,
            width=width,
            height=height
        )
        img = result.images[0]
        gen_time = time.time() - t0
        print(f"Image {i+1}/{n} generated in {gen_time:.2f}s")
        
        filename = f"{uuid.uuid4().hex}.jpg"
        filepath = os.path.join(IMAGE_DIR, filename)
        img.save(filepath, format="JPEG", quality=90)
        
        if req.response_format == "url":
            data_items.append(ImageDataItem(
                url=f"http://{host}/images/{filename}",
                revised_prompt=req.prompt
            ))
        else:
            buffer = io.BytesIO()
            img.save(buffer, format="JPEG", quality=90)
            b64_str = base64.b64encode(buffer.getvalue()).decode("utf-8")
            data_items.append(ImageDataItem(
                b64_json=b64_str,
                url=f"http://{host}/images/{filename}",
                revised_prompt=req.prompt
            ))
            
    return ImageGenerationResponse(
        created=int(time.time()),
        data=data_items
    )

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
