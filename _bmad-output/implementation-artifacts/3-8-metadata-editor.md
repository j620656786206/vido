# Story 3.8: Metadata Editor

Status: review

## Story

As a **media collector**,
I want to **manually edit metadata for any media item**,
So that **I can correct errors or add missing information**.

## Acceptance Criteria

1. **AC1: Edit Form with All Fields**
   - Given a media item in the library
   - When the user clicks "Edit Metadata"
   - Then an edit form opens with all editable fields:
     - Title (Chinese/English)
     - Year
     - Genre
     - Director
     - Cast
     - Description
     - Poster (upload or URL)

2. **AC2: Persist Changes**
   - Given the user modifies metadata
   - When saving changes
   - Then the changes are persisted to the database
   - And the source is updated to "Manual"

3. **AC3: Custom Poster Upload**
   - Given the user uploads a custom poster
   - When the upload completes
   - Then the image is resized and optimized
   - And stored in local cache

4. **AC4: Form Validation**
   - Given the edit form is open
   - When the user submits with invalid data
   - Then validation errors are shown inline
   - And the form is not submitted until errors are fixed

## Tasks / Subtasks

- [x] Task 1: Create Metadata Update API Endpoint (AC: 2)
  - [x] 1.1: Create `PUT /api/v1/media/{id}/metadata` endpoint
  - [x] 1.2: Accept all editable metadata fields
  - [x] 1.3: Validate required fields (title, year)
  - [x] 1.4: Update metadata source to "manual"
  - [x] 1.5: Write handler tests

- [x] Task 2: Create Poster Upload API Endpoint (AC: 3)
  - [x] 2.1: Create `POST /api/v1/media/{id}/poster` endpoint
  - [x] 2.2: Accept multipart/form-data image upload
  - [x] 2.3: Validate file type (jpg, png, webp)
  - [x] 2.4: Validate file size (max 5MB)
  - [x] 2.5: Write handler tests

- [x] Task 3: Implement Image Processing (AC: 3)
  - [x] 3.1: Create `/apps/api/internal/images/processor.go`
  - [x] 3.2: Resize images to standard poster dimensions (300x450)
  - [x] 3.3: Optimize image quality (80% JPEG, WebP conversion)
  - [x] 3.4: Generate thumbnail (100x150)
  - [x] 3.5: Store in local cache directory
  - [x] 3.6: Write processing tests

- [x] Task 4: Create Metadata Edit Service (AC: 2, 3)
  - [x] 4.1: Create `MetadataEditService` in services package
  - [x] 4.2: Implement `UpdateMetadata()` method
  - [x] 4.3: Implement `UploadPoster()` method
  - [x] 4.4: Handle poster URL fetch (if URL provided instead of upload)
  - [x] 4.5: Write service tests

- [x] Task 5: Create Metadata Editor Dialog Component (AC: 1, 4)
  - [x] 5.1: Create `MetadataEditorDialog.tsx` component
  - [x] 5.2: Implement form with all editable fields
  - [x] 5.3: Add react-hook-form for form management
  - [x] 5.4: Add zod for validation schema
  - [x] 5.5: Write component tests

- [x] Task 6: Create Form Field Components (AC: 1, 4)
  - [x] 6.1: Create `TitleField.tsx` (Chinese/English toggle)
  - [x] 6.2: Create `GenreSelector.tsx` (multi-select)
  - [x] 6.3: Create `CastEditor.tsx` (add/remove actors)
  - [x] 6.4: Create `PosterUploader.tsx` (drag-drop, URL input)
  - [x] 6.5: Write component tests for each

- [x] Task 7: Implement Poster Upload UI (AC: 3)
  - [x] 7.1: Add drag-and-drop zone
  - [x] 7.2: Add file picker button
  - [x] 7.3: Add URL input option
  - [x] 7.4: Show upload progress
  - [x] 7.5: Show preview after upload
  - [x] 7.6: Write upload tests

- [x] Task 8: Integration with Media Detail Page (AC: 1)
  - [x] 8.1: Add "Edit Metadata" button to media detail page
  - [x] 8.2: Open editor dialog on click
  - [x] 8.3: Refresh detail page after save
  - [x] 8.4: Show success toast notification

## Dev Notes

### Architecture Requirements

**FR21: Manually edit media metadata**
- Form validation for required fields
- Image processing for poster optimization

### API Design

**Update Metadata Endpoint:**
```
PUT /api/v1/media/{id}/metadata
Content-Type: application/json

{
  "title": "鬼滅之刃",
  "titleEnglish": "Demon Slayer",
  "year": 2019,
  "genres": ["動作", "奇幻", "冒險"],
  "director": "外崎春雄",
  "cast": ["花江夏樹", "鬼頭明里", "下野紘"],
  "overview": "大正時代的日本，善良的少年炭治郎...",
  "posterUrl": "https://..." // Optional: if using URL instead of upload
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "media-123",
    "title": "鬼滅之刃",
    "metadataSource": "manual",
    "updatedAt": "2026-01-18T12:00:00Z"
  }
}
```

**Poster Upload Endpoint:**
```
POST /api/v1/media/{id}/poster
Content-Type: multipart/form-data

file: <binary image data>
```

**Response:**
```json
{
  "success": true,
  "data": {
    "posterUrl": "/posters/media-123.webp",
    "thumbnailUrl": "/posters/media-123-thumb.webp"
  }
}
```

### Image Processing Specifications

| Dimension | Size | Format | Quality |
|-----------|------|--------|---------|
| Full poster | 300x450 | WebP | 80% |
| Thumbnail | 100x150 | WebP | 75% |
| Max upload | - | jpg/png/webp | - |
| Max file size | 5MB | - | - |

```go
// ImageProcessor handles poster image optimization
type ImageProcessor struct {
    cacheDir string
    logger   *slog.Logger
}

type ProcessedImage struct {
    PosterPath    string
    ThumbnailPath string
    OriginalSize  int64
    ProcessedSize int64
}

func (p *ImageProcessor) ProcessPoster(input io.Reader, mediaID string) (*ProcessedImage, error) {
    // 1. Decode image
    img, format, err := image.Decode(input)
    if err != nil {
        return nil, fmt.Errorf("failed to decode image: %w", err)
    }

    // 2. Resize to poster dimensions (300x450)
    poster := resize.Resize(300, 450, img, resize.Lanczos3)

    // 3. Create thumbnail (100x150)
    thumbnail := resize.Resize(100, 150, img, resize.Lanczos3)

    // 4. Encode to WebP
    posterPath := filepath.Join(p.cacheDir, fmt.Sprintf("%s.webp", mediaID))
    thumbPath := filepath.Join(p.cacheDir, fmt.Sprintf("%s-thumb.webp", mediaID))

    // 5. Save files
    // ...

    return &ProcessedImage{
        PosterPath:    posterPath,
        ThumbnailPath: thumbPath,
    }, nil
}
```

### Frontend Component Design

**MetadataEditorDialog.tsx:**
```tsx
interface MetadataEditorDialogProps {
  isOpen: boolean;
  onClose: () => void;
  mediaId: string;
  initialData: MediaMetadata;
  onSuccess: () => void;
}

const metadataSchema = z.object({
  title: z.string().min(1, "標題為必填"),
  titleEnglish: z.string().optional(),
  year: z.number().min(1900).max(2100),
  genres: z.array(z.string()),
  director: z.string().optional(),
  cast: z.array(z.string()),
  overview: z.string().optional(),
});

type MetadataFormData = z.infer<typeof metadataSchema>;

const MetadataEditorDialog: React.FC<MetadataEditorDialogProps> = ({
  isOpen,
  onClose,
  mediaId,
  initialData,
  onSuccess,
}) => {
  const form = useForm<MetadataFormData>({
    resolver: zodResolver(metadataSchema),
    defaultValues: initialData,
  });

  const updateMutation = useUpdateMetadata();
  const uploadMutation = useUploadPoster();

  const onSubmit = async (data: MetadataFormData) => {
    await updateMutation.mutateAsync({ mediaId, ...data });
    onSuccess();
    onClose();
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>編輯媒體資訊</DialogTitle>
        </DialogHeader>

        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <TitleField form={form} />
          <YearField form={form} />
          <GenreSelector form={form} />
          <DirectorField form={form} />
          <CastEditor form={form} />
          <OverviewField form={form} />
          <PosterUploader
            mediaId={mediaId}
            currentPoster={initialData.posterUrl}
            onUpload={uploadMutation.mutate}
          />

          <DialogFooter>
            <Button variant="outline" onClick={onClose}>取消</Button>
            <Button type="submit" disabled={updateMutation.isPending}>
              {updateMutation.isPending ? "儲存中..." : "儲存"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
```

**PosterUploader.tsx:**
```tsx
interface PosterUploaderProps {
  mediaId: string;
  currentPoster?: string;
  onUpload: (file: File) => void;
}

const PosterUploader: React.FC<PosterUploaderProps> = ({
  mediaId,
  currentPoster,
  onUpload,
}) => {
  const [preview, setPreview] = useState<string | null>(currentPoster || null);
  const [uploadMethod, setUploadMethod] = useState<'file' | 'url'>('file');

  const onDrop = useCallback((acceptedFiles: File[]) => {
    const file = acceptedFiles[0];
    if (file) {
      setPreview(URL.createObjectURL(file));
      onUpload(file);
    }
  }, [onUpload]);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: { 'image/*': ['.jpg', '.jpeg', '.png', '.webp'] },
    maxSize: 5 * 1024 * 1024, // 5MB
  });

  return (
    <div className="space-y-2">
      <Label>海報圖片</Label>
      <Tabs value={uploadMethod} onValueChange={setUploadMethod}>
        <TabsList>
          <TabsTrigger value="file">上傳檔案</TabsTrigger>
          <TabsTrigger value="url">輸入網址</TabsTrigger>
        </TabsList>

        <TabsContent value="file">
          <div
            {...getRootProps()}
            className={cn(
              "border-2 border-dashed rounded-lg p-6 text-center cursor-pointer",
              isDragActive && "border-primary bg-primary/5"
            )}
          >
            <input {...getInputProps()} />
            {preview ? (
              <img src={preview} alt="Preview" className="max-h-48 mx-auto" />
            ) : (
              <p>拖放圖片或點擊選擇檔案</p>
            )}
          </div>
        </TabsContent>

        <TabsContent value="url">
          <Input placeholder="輸入海報圖片網址" />
        </TabsContent>
      </Tabs>
    </div>
  );
};
```

### Genre Options (Reference)

```typescript
const GENRE_OPTIONS = [
  { value: "action", label: "動作" },
  { value: "adventure", label: "冒險" },
  { value: "animation", label: "動畫" },
  { value: "comedy", label: "喜劇" },
  { value: "crime", label: "犯罪" },
  { value: "documentary", label: "紀錄片" },
  { value: "drama", label: "劇情" },
  { value: "family", label: "家庭" },
  { value: "fantasy", label: "奇幻" },
  { value: "history", label: "歷史" },
  { value: "horror", label: "恐怖" },
  { value: "music", label: "音樂" },
  { value: "mystery", label: "懸疑" },
  { value: "romance", label: "愛情" },
  { value: "sci-fi", label: "科幻" },
  { value: "thriller", label: "驚悚" },
  { value: "war", label: "戰爭" },
  { value: "western", label: "西部" },
];
```

### Project Structure Notes

**Backend Files to Create:**
```
/apps/api/internal/handlers/
└── metadata_edit_handler.go

/apps/api/internal/services/
└── metadata_edit_service.go

/apps/api/internal/images/
├── processor.go
└── processor_test.go
```

**Frontend Files to Create:**
```
/apps/web/src/components/metadata/
├── MetadataEditorDialog.tsx
├── MetadataEditorDialog.spec.tsx
├── TitleField.tsx
├── GenreSelector.tsx
├── CastEditor.tsx
├── PosterUploader.tsx
├── PosterUploader.spec.tsx
└── index.ts
```

### Testing Strategy

**Backend Tests:**
1. Metadata update handler tests
2. Poster upload handler tests
3. Image processor tests (resize, optimize)
4. Service integration tests

**Frontend Tests:**
1. Form validation tests
2. Poster uploader tests (file, URL)
3. Form submission tests
4. Error handling tests

**Coverage Targets:**
- Backend handlers: ≥70%
- Backend services: ≥80%
- Image processor: ≥80%
- Frontend components: ≥70%

### Error Codes

Following project-context.md Rule 7:
- `METADATA_UPDATE_FAILED` - Failed to update metadata
- `POSTER_UPLOAD_FAILED` - Failed to upload poster
- `POSTER_INVALID_FORMAT` - Invalid image format
- `POSTER_TOO_LARGE` - Image exceeds 5MB limit
- `VALIDATION_REQUIRED_FIELD` - Required field missing

### Dependencies

**Go Libraries:**
- `github.com/nfnt/resize` - Image resizing
- `github.com/chai2010/webp` - WebP encoding

**Frontend Libraries:**
- `react-hook-form` - Form management
- `zod` - Validation schema
- `react-dropzone` - File upload

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.8]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR21]
- [Source: project-context.md#Rule-5-TanStack-Query]
- [Source: project-context.md#Rule-4-Layered-Architecture]

### Previous Story Intelligence

**From Story 3.7 (Manual Search):**
- Dialog component pattern established
- TanStack Query mutation patterns available
- API response format consistent

**From Epic 2 (Media Detail):**
- Media detail page exists for integration
- Metadata display patterns established

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

### Completion Notes List

- All 8 tasks completed for Story 3.8
- Backend: Metadata update and poster upload API endpoints with image processing
- Frontend: MetadataEditorDialog with form validation using react-hook-form and zod
- Form components: GenreSelector, CastEditor, PosterUploader (with drag-drop)
- Integration: Edit button on media detail page with success toast

### File List

**Backend (Go):**
- `apps/api/internal/handlers/metadata_handler.go` - UpdateMetadata, UploadPoster handlers
- `apps/api/internal/handlers/metadata_handler_test.go` - Handler tests
- `apps/api/internal/services/metadata_service.go` - Interface and types
- `apps/api/internal/services/metadata_edit_service.go` - MetadataEditService implementation
- `apps/api/internal/services/metadata_edit_service_test.go` - Service tests
- `apps/api/internal/images/processor.go` - Image resizing and optimization
- `apps/api/internal/images/processor_test.go` - Image processor tests

**Frontend (TypeScript/React):**
- `apps/web/src/services/metadata.ts` - API client with updateMetadata, uploadPoster
- `apps/web/src/hooks/useMetadataEditor.ts` - useUpdateMetadata, useUploadPoster hooks
- `apps/web/src/components/metadata-editor/index.ts` - Barrel exports
- `apps/web/src/components/metadata-editor/MetadataEditorDialog.tsx` - Main dialog component
- `apps/web/src/components/metadata-editor/MetadataEditorDialog.spec.tsx` - Dialog tests (14 tests)
- `apps/web/src/components/metadata-editor/GenreSelector.tsx` - Genre multi-select component
- `apps/web/src/components/metadata-editor/GenreSelector.spec.tsx` - GenreSelector tests (6 tests)
- `apps/web/src/components/metadata-editor/CastEditor.tsx` - Cast add/remove component
- `apps/web/src/components/metadata-editor/CastEditor.spec.tsx` - CastEditor tests (9 tests)
- `apps/web/src/components/metadata-editor/PosterUploader.tsx` - Poster upload with drag-drop
- `apps/web/src/components/metadata-editor/PosterUploader.spec.tsx` - PosterUploader tests (13 tests)
- `apps/web/src/routes/media/$type.$id.tsx` - Media detail page with Edit button integration

## Change Log

| Date | Change | Author |
|------|--------|--------|
| 2026-01-25 | Story implementation complete - all 8 tasks done, 42 frontend + all backend tests pass | Claude Opus 4.5 |
| 2026-01-25 | Status updated to review | Claude Opus 4.5 |
