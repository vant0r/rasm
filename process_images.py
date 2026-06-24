#!/usr/bin/env python3
"""
Process all Bilet images:
1. Crop only the illustration area (remove status bar, buttons, text)
2. Add Vatanparvar logo in the bottom-right corner to cover watermarks
"""

from PIL import Image
import os
import glob

# Crop boundaries for the illustration area (consistent across all 1080x2460 screenshots)
CROP_TOP = 414
CROP_BOTTOM = 1133
CROP_LEFT = 36
CROP_RIGHT = 1039

# Logo size (relative to cropped image height)
LOGO_SIZE_RATIO = 0.18  # 18% of the cropped image height

# Logo position offset from bottom-right corner
LOGO_MARGIN_RIGHT = 10
LOGO_MARGIN_BOTTOM = 10


def process_images():
    # Load logo
    logo_path = os.path.join(os.path.dirname(__file__), 'processor', 'logo.png')
    if not os.path.exists(logo_path):
        print(f"Error: Logo not found at {logo_path}")
        return

    logo_orig = Image.open(logo_path).convert('RGBA')
    print(f"Logo loaded: {logo_orig.size}")

    # Process each Bilet folder
    folders = sorted(glob.glob(os.path.join(os.path.dirname(__file__), 'Bilet *')))
    
    total_processed = 0
    
    for folder in folders:
        folder_name = os.path.basename(folder)
        print(f"\nProcessing {folder_name}...")
        
        # Get all jpg files in the folder
        images = sorted(glob.glob(os.path.join(folder, '*.jpg')))
        
        for img_path in images:
            try:
                # Open image
                img = Image.open(img_path).convert('RGB')
                width, height = img.size
                
                # Verify expected dimensions
                if width != 1080 or height != 2460:
                    print(f"  WARNING: {os.path.basename(img_path)} has unexpected size {width}x{height}, skipping")
                    continue
                
                # Crop the illustration area
                cropped = img.crop((CROP_LEFT, CROP_TOP, CROP_RIGHT, CROP_BOTTOM))
                crop_width, crop_height = cropped.size
                
                # Calculate logo size
                logo_size = int(crop_height * LOGO_SIZE_RATIO)
                logo_resized = logo_orig.resize((logo_size, logo_size), Image.LANCZOS)
                
                # Calculate logo position (bottom-right corner)
                logo_x = crop_width - logo_size - LOGO_MARGIN_RIGHT
                logo_y = crop_height - logo_size - LOGO_MARGIN_BOTTOM
                
                # Paste logo onto cropped image
                # Convert cropped to RGBA for compositing
                cropped_rgba = cropped.convert('RGBA')
                cropped_rgba.paste(logo_resized, (logo_x, logo_y), logo_resized)
                
                # Convert back to RGB for JPEG saving
                final = cropped_rgba.convert('RGB')
                
                # Save back to the same path (overwrite original)
                final.save(img_path, 'JPEG', quality=95)
                total_processed += 1
                print(f"  Processed: {os.path.basename(img_path)}")
                
            except Exception as e:
                print(f"  ERROR processing {os.path.basename(img_path)}: {e}")
    
    print(f"\n{'='*50}")
    print(f"Done! Total images processed: {total_processed}")


if __name__ == '__main__':
    process_images()
