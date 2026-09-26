# Cloud Run 部署手冊 — 2026-10-06 執行版

狀態：草案，供 10/6 GCP 專案／billing／`GEMINI_API_KEY` 到位當天照做
更新：2026-09-26
前置演練：本機 `docker build` + `docker run` + curl，見〈本次本地演練結果〉一節；**未執行任何 `gcloud` 寫入動作、未 push image、未建立任何雲端資源**

> 這份手冊記錄的是「10/6 當天要做什麼」。既有的 [`CLOUD_RUN_BASELINE.md`](CLOUD_RUN_BASELINE.md) 是穩態容器契約文件（環境變數表、smoke check 定義），這份是一次性的執行清單，兩者對照著看；本手冊完成後，`CLOUD_RUN_BASELINE.md` 不需要改，但 `evidence/EB-008`、`evidence/EB-009`、`README.md:9` 的宣告值需要回填（見下方對應段落）。

## 0. 這次演練實際查證了什麼

讀了 `scripts/deploy-cloud-run.sh`、`Dockerfile`、`cloudbuild.yaml`、`scripts/bootstrap-w1.sh`、`scripts/record-promotion.sh`、`cmd/context-rail/main.go`、`internal/change/advisor.go`、`internal/change/service.go`、`.github/workflows/ci.yml`、`docs/operations/CLOUD_RUN_BASELINE.md`、`docs/operations/REAL_RUN_PLAYBOOK.md`。本機用 `docker build --platform=linux/amd64` 建置 image、`docker run` 起容器、curl 打 `/healthz`、`/`、`/v1/projects`、`POST /v1/projects`，全部符合預期（見第 5 節）。**沒有**安裝 `gcloud`（本機 `command not found: gcloud`），因此 Artifact Registry／Cloud Build／Cloud Run／Secret Manager 相關指令全部**未實測**，旗標是否完全正確以 10/6 當天實跑為準。

---

## 1. 前置清單 — 10/6 需要 William 提供／建立的 GCP 值

| 值 | 用途 | 出處 | 目前狀態 |
| --- | --- | --- | --- |
| GCP 專案 id（`PROJECT_ID`） | 所有資源的容器；`scripts/deploy-cloud-run.sh` 第一個必填參數 | `scripts/deploy-cloud-run.sh:30`（`PROJECT_ID="${1:?usage: ...}"`）、`scripts/bootstrap-w1.sh:18` | 未建立；`bootstrap-w1.sh:78` 若專案不存在會用 `gcloud projects create` 建立，名稱寫死 `ContextRail` |
| Billing account 已連結該專案 | `gcloud services enable` 與 Cloud Build 都需要 billing | `scripts/bootstrap-w1.sh:82-97`（讀 `gcloud billing accounts list --filter=open=true`，只有剛好一個帳號時自動連結，否則印出來要求手動 `gcloud billing projects link`） | 待 10/6 |
| Region（預設 `asia-east1`） | Artifact Registry、Cloud Run 部署區域 | `scripts/deploy-cloud-run.sh:31`、`cloudbuild.yaml:12`、`scripts/bootstrap-w1.sh:19` | 有預設值，可延用 |
| Cloud Run service 名稱（預設 `context-rail-staging`） | staging 服務名稱 | `scripts/deploy-cloud-run.sh:32` | 有預設值 |
| Artifact Registry repo 名稱（寫死 `context-rail`） | 存放 image | `scripts/deploy-cloud-run.sh:33`、`cloudbuild.yaml:13` | 腳本自動 `describe` 沒有就 `create`（`scripts/deploy-cloud-run.sh:49-51`） |
| 部署身份（操作機器上 `gcloud auth login` 的帳號） | 需要能 enable API、建 Artifact Registry repo、跑 Cloud Build、部署 Cloud Run | `scripts/deploy-cloud-run.sh:35`（只檢查 `command -v gcloud`，**沒有檢查權限**）、`scripts/bootstrap-w1.sh:70-74` | **未定義**——目前所有腳本都假設「當前 gcloud 使用者已有夠大的權限」，沒有指定或建立專用部署用 service account。10/6 若用 William 個人帳號（Owner/Editor）最省事，但要留意這代表本地開發機持有專案級高權限憑證 |
| Cloud Run **執行期**服務帳號（runtime SA） | 容器實際跑起來後用什麼身份呼叫 GCP API（目前程式碼完全不呼叫其他 GCP API，但要讀 Secret Manager 的 `GEMINI_API_KEY` 需要這個身份有 `roles/secretmanager.secretAccessor`） | `cloudbuild.yaml:47-55` 的 `gcloud run deploy` **沒有 `--service-account` 旗標** | **未指定** → 會落到該專案的 Compute Engine 預設服務帳號（`<PROJECT_NUMBER>-compute@developer.gserviceaccount.com`），權限通常比需要的寬。建議 10/6 建立一個最小權限的專用 SA（見第 2 節步驟 3） |
| `GEMINI_API_KEY` 的值 | 啟用 `internal/change/advisor.go` 的 `GeminiAdvisor`；不設就 fallback 成 `RuleAdvisor` | `cmd/context-rail/main.go:34,150-152`、`internal/change/advisor.go:148-149`（`"GEMINI_API_KEY not configured"`）、`docs/operations/CLOUD_RUN_BASELINE.md:35` | William 提供；**目前沒有任何腳本把它放進 Secret Manager 或部署指令**（見第 3 節「本次演練發現的落差」） |
| `CONTEXT_RAIL_GEMINI_MODEL`（可選） | 覆寫預設 model | `cmd/context-rail/main.go:35`、`internal/change/advisor.go:134-136`（預設 `gemini-2.0-flash`） | 有預設值，可不設 |
| `GITHUB_TOKEN`（可選） | 啟用 candidate 的 GitHub read-back（compare / PR reviews / check runs） | `cmd/context-rail/main.go:36`、`docs/operations/CLOUD_RUN_BASELINE.md:37` | 本次任務範圍**不需要**——demo 用固定 fixture，不牽涉真實 repo read-back；除非 10/6 要順便展示 `REAL_RUN_PLAYBOOK.md` 的流程 |
| Firestore／Storage | 目前**完全沒有用到** | 全 repo 搜尋 `firestore`／`storage` 只有一處註解：`internal/topology/store.go:23`（`// persistence decision (files first, Firestore later)` ——純規劃註記，非程式碼依賴） | 不需要 10/6 準備任何 Firestore/Storage 資源 |
| 容器內部狀態目錄（不是 GCP 值，是風險提醒） | Governance ledger（Change / Candidate / Release / Topology 版本）存放位置 | `Dockerfile:34-41`（`CONTEXT_RAIL_STATE_DIR=/tmp/context-rail-state`，註解明講「不是持久化儲存」） | Cloud Run instance 重啟／scale-to-zero／重新部署都會清空；`cloudbuild.yaml:53` 目前 `--min-instances=0`，demo 過程中閒置太久有被 Google 回收、狀態消失的風險（見第 6 節風險） |

---

## 2. 逐步指令（10/6 當天）

以下每一步都附**預期輸出**與**失敗時的處置**。標「未實測」的旗標是照著 `cloudbuild.yaml` / `deploy-cloud-run.sh` 既有內容或 GCP 官方文件推導，10/6 當天以實際輸出為準，不要假裝已經驗證過。

### 步驟 0 — 安裝與登入（操作機器）

```sh
brew install --cask google-cloud-sdk   # 或既有安裝
gcloud auth login
gcloud auth application-default login  # Cloud Build / client library 認證用
```

**預期**：瀏覽器登入完成，`gcloud auth list --filter=status:ACTIVE --format='value(account)'` 印出登入帳號。
**失敗處置**：登入失敗多半是網路或帳號問題，重跑 `gcloud auth login`；不要在無人值守腳本裡塞帳密。

### 步驟 1 — 建立／選定專案與 billing

可直接用既有腳本（`scripts/bootstrap-w1.sh:69-97`）跑到 GCP 這段，或手動：

```sh
gcloud projects create "${PROJECT_ID}" --name="ContextRail" --set-as-default   # 專案不存在才需要
gcloud config set project "${PROJECT_ID}"
gcloud billing accounts list --filter=open=true
gcloud billing projects link "${PROJECT_ID}" --billing-account="<ACCOUNT_ID>"
```

**預期**：`gcloud billing projects describe "${PROJECT_ID}" --format='value(billingEnabled)'` 回傳 `True`。
**失敗處置**：billing account 不只一個時 `bootstrap-w1.sh:88-96` 會直接印出清單並退出，需要人工指定 `--billing-account`；沒有任何開通的 billing account 就必須先在 Cloud Console 開一個，這不是腳本能自動做的。

### 步驟 2 — 啟用 API、建 Artifact Registry repo

```sh
gcloud services enable run.googleapis.com cloudbuild.googleapis.com artifactregistry.googleapis.com secretmanager.googleapis.com
gcloud artifacts repositories describe context-rail --location="${REGION}" \
  || gcloud artifacts repositories create context-rail --location="${REGION}" \
       --repository-format=docker --description="ContextRail images"
```

對應 `scripts/deploy-cloud-run.sh:46-51`，唯一差異是**多加了 `secretmanager.googleapis.com`**——原腳本沒有這行，是本次演練發現的落差（原腳本從沒打算讀 Secret Manager）。
**預期**：`describe` 回傳 repo 詳細資訊，或 `create` 成功建立。
**失敗處置**：權限不足（`PERMISSION_DENIED`）代表步驟 0 的帳號沒有 `roles/artifactregistry.admin` 或等效權限；換帳號或加角色。

### 步驟 3 — 建立最小權限的 Cloud Run 執行身份（新步驟，原腳本沒有）

原本 `cloudbuild.yaml:47-55` 的 `gcloud run deploy` 沒有 `--service-account`，會用專案預設的 Compute Engine SA。若要讓 `GEMINI_API_KEY` 走 Secret Manager 而不是明文塞進環境變數，需要一個有 `secretAccessor` 權限的執行身份：

```sh
gcloud iam service-accounts create context-rail-run \
  --display-name="ContextRail Cloud Run runtime"
gcloud secrets create gemini-api-key --replication-policy=automatic   # 若尚未建立
echo -n "${GEMINI_API_KEY}" | gcloud secrets versions add gemini-api-key --data-file=-
gcloud secrets add-iam-policy-binding gemini-api-key \
  --member="serviceAccount:context-rail-run@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

**未實測**：以上 IAM/Secret Manager 指令是依 GCP 官方文件慣例寫的，10/6 當天第一次跑務必看清楚錯誤訊息，不要照抄後假設成功。
**失敗處置**：`gemini-api-key` 密鑰值請勿用 shell history 留痕（用 `echo -n ... | gcloud secrets versions add` 且該行結束後 `history -d` 或直接不留在互動 shell）；若 secret 已存在，`create` 會報 `ALREADY_EXISTS`，改用 `gcloud secrets versions add` 即可。

### 步驟 4 — 修改部署指令，帶入 secret（原腳本落差的實際修法）

`cloudbuild.yaml:47-55` 目前的 `gcloud run deploy` 沒有任何 `--set-secrets` 或 `--service-account`。10/6 當天需要手動加這兩個旗標再跑（或先改 `cloudbuild.yaml` 再 commit）：

```sh
gcloud run deploy context-rail-staging \
  --image="${IMAGE}@${DIGEST}" \
  --region="${REGION}" \
  --platform=managed \
  --port=8080 \
  --allow-unauthenticated \
  --min-instances=0 --max-instances=2 \
  --memory=256Mi --cpu=1 \
  --service-account="context-rail-run@${PROJECT_ID}.iam.gserviceaccount.com" \
  --set-secrets="GEMINI_API_KEY=gemini-api-key:latest" \
  --labels=context-rail-environment=staging,context-rail-commit="${SHORT_SHA}"
```

**未實測**：`--set-secrets` 語法（`ENV_VAR=SECRET_NAME:VERSION`）依 `gcloud run deploy` 官方文件；10/6 當天第一次跑務必核對 `gcloud run services describe context-rail-staging --region="${REGION}" --format=yaml` 裡真的看得到這個環境變數是從 secret 掛載，不是空的。
**如果懶得改 `cloudbuild.yaml`**：也可以先跑 `scripts/deploy-cloud-run.sh`（這時 Gemini 還沒生效，會 fallback 成 rule-advisor），部署完再用 `gcloud run services update context-rail-staging --region="${REGION}" --update-secrets=GEMINI_API_KEY=gemini-api-key:latest --service-account=...` 補上，效果一樣，只是多一次 revision。

### 步驟 5 — Build / Push / Deploy（照既有腳本）

```sh
scripts/deploy-cloud-run.sh "${PROJECT_ID}" "${REGION}" context-rail-staging
```

這一行等同跑完 `scripts/deploy-cloud-run.sh:43-68` 全部內容：`gcloud config set project` → enable API → ensure AR repo → `gcloud builds submit --config cloudbuild.yaml`（雲端建置，天然是 linux/amd64，不受本機 Apple Silicon 影響）→ 讀回 `url` / `revision` / `image` → 跑 `smoke()`（`scripts/deploy-cloud-run.sh:14-23`：`/healthz` 要含 `"status":"ok"`、`/v1/projects` 要有 ≥2 個 fixture project、`/` 要 200、`POST /v1/projects` 要 405）。

若已經照步驟 3-4 改過 `cloudbuild.yaml`（把 `--service-account` 和 `--set-secrets` 寫進 `deploy-staging` 這個 build step），這行會直接把 secret 帶進去；若沒改 `cloudbuild.yaml`，就要在這行跑完後補步驟 4 的 `gcloud run services update`。

**預期輸出**（`scripts/deploy-cloud-run.sh:61-70`）：

```
== deployed
url:      https://context-rail-staging-<hash>-<region>.a.run.app
revision: context-rail-staging-00001-xxx
image:    <region>-docker.pkg.dev/<project>/context-rail/context-rail@sha256:<digest>
commit:   <short-sha>
```

**失敗處置**：
- Cloud Build 失敗最常見是 `npm ci` 或 `go build` 在雲端環境差異（例如 lockfile 不同步）——先看 Cloud Build log（`gcloud builds log <BUILD_ID>`）。
- `gcloud run deploy` 失敗且訊息含 `PERMISSION_DENIED` on secret：步驟 3 的 IAM binding 沒生效或打錯 SA email。
- smoke 失敗在 `/healthz`：容器沒有正確監聽 `$PORT`（理論上不會，`cmd/context-rail/main.go:207-214` 的 `listenAddress` 已經處理，本次本地演練也驗證過）；失敗在 `/v1/projects` 缺 fixture：image 沒把 `demo/order-operations-portal` / `examples/support-insights` 複製進去（檢查 `.dockerignore` 有沒有誤擋，見第 6 節）。

### 步驟 6 — 驗證 Gemini 真的被呼叫到（腳本沒有的驗證，需手動跑）

`scripts/deploy-cloud-run.sh` 的 smoke check **不驗證 Gemini**，只驗證基本 API。程式碼裡有沒有走 Gemini 只能從實際回應內容看：

```sh
BASE="https://context-rail-staging-<hash>-<region>.a.run.app"
curl -fsS -X POST "${BASE}/v1/projects/order-operations-portal/changes" \
  -H 'Content-Type: application/json' -H 'X-ContextRail-Actor: william' \
  -d '{"title":"smoke test change","objective":"verify gemini advisor wiring","target_environment":"staging","data_classification":"internal"}' \
  | python3 -m json.tool
```

**預期**：若 `GEMINI_API_KEY` 真的生效，回應裡每個 option 的 `"advisor_source"` 欄位（對應 `internal/change/model.go:115`）應該是 `"gemini:gemini-2.0-flash"`（或所設定的 model，見 `internal/change/advisor.go:140`），不是 `"rule-advisor"`。若 Gemini 呼叫失敗（key 錯、網路不通、額度用盡），會**靜默 fallback**，`evaluation.observations` 裡會多一條 `"advisor gemini:<model> unavailable (<err>); rule-based candidates used"`（`internal/change/service.go:467-471`）——這條訊息就是唯一能分辨「fallback 了」與「本來就沒設 key」的地方，10/6 當天務必檢查這個欄位，不要只看 HTTP 200 就當作 Gemini 有生效。
**失敗處置**：如果一直 fallback，先確認 secret 掛載進容器的環境變數真的叫 `GEMINI_API_KEY`（`gcloud run services describe ... --format=yaml` 看 `env` 區塊），再確認 key 本身在 Google AI Studio 是 active 且沒有地區限制。

### 步驟 7 — 重跑 smoke（單獨驗證，不必重新部署）

```sh
scripts/deploy-cloud-run.sh --smoke "${BASE}"
```

對應 `scripts/deploy-cloud-run.sh:14-27`。

---

## 3. 本次演練發現的落差（原腳本沒做但 10/6 需要的事）

1. **`GEMINI_API_KEY` 沒有任何自動化路徑進 Cloud Run**——`cloudbuild.yaml:41-57` 的 `deploy-staging` step 完全沒有 `--set-secrets` / `--update-env-vars`，`scripts/deploy-cloud-run.sh` 也没有相關參數。`docs/operations/CLOUD_RUN_BASELINE.md:35` 只是文字說明「應該設成 secret」，程式碼從未真的做。→ 本手冊第 2 節步驟 3-4 補了具體指令，**未實測**。
2. **Cloud Run 執行身份未指定** → 落到專案預設 Compute SA，權限範圍不明確、也拿不到 Secret Manager 存取權。→ 本手冊補了建立專用 SA 的步驟。
3. **`secretmanager.googleapis.com` 沒有被 enable**（`scripts/deploy-cloud-run.sh:46` 只 enable 三個 API）。
4. 這三點都是**新增步驟**，不是修改既有腳本的 bug；沒有動 `scripts/deploy-cloud-run.sh` / `cloudbuild.yaml` 本身，因為改雲端部署腳本超出「本地建置演練」的授權範圍，且沒有實際環境驗證過改動是否正確，貿然改腳本比手動補步驟風險更高。

## 4. 部署後要回填的證據

| 檔案 | 現況（宣告值） | 10/6 之後要填的真實值 |
| --- | --- | --- |
| `README.md:9` | `` `<deployed Cloud Run URL — filled in at submission>` `` | 真實 Cloud Run URL |
| `evidence/EB-008/README.md:28` | 「**No real deployment.** Every deployment record in evidence is declared.」 | 改成指向真實 revision / digest 的記錄，並附上 `scripts/record-promotion.sh` 實際輸出（該腳本本身這次沒有跑，它是給*受管理的 demo application*〔`order-operations-portal`〕記錄部署用的，不是給 ContextRail 自己） |
| `evidence/EB-009/README.md:35` | 同上一句（prod-demo 版本） | 同上，另外要有 prod-demo 服務自己的 revision/digest |
| `internal/change/advisor.go:124-125` 附近的程式碼註解「NOT exercised... treat as UNVERIFIED until a run with GEMINI_API_KEY is recorded as evidence」 | UNVERIFIED | 10/6 跑完步驟 6 後，把該次 HTTP 回應（`advisor_source: "gemini:..."`）存成新的 evidence bundle，並可以考慮把這句註解更新或移除 UNVERIFIED 標記（程式碼註解本身要不要改由 William 決定，這次沒有動它） |
| `decisions/DR-008-staging-promotion.json` / `DR-009-prod-demo-promotion.json` | `ACCEPTED_FOR_DEVELOPMENT` | 是否要升級決策狀態由 William 判斷，不在本次任務範圍內 |

## 5. 回滾方式

Cloud Run 對每次 `gcloud run deploy` 都會產生新 revision，不會覆蓋舊的（除非手動刪除），因此回滾是流量切回舊 revision：

```sh
gcloud run services describe context-rail-staging --region="${REGION}" \
  --format='value(status.traffic)'                      # 列出目前流量分配與 revision 清單
gcloud run services update-traffic context-rail-staging --region="${REGION}" \
  --to-revisions="<OLD_REVISION>=100"
```

**未實測**——`update-traffic` 語法依 GCP 官方文件；沒有在這次演練裡驗證過任何 revision 切換。若要完全撤回（例如誤設了錯誤的 secret 導致服務起不來），可以直接刪除新 revision：`gcloud run revisions delete <NEW_REVISION> --region="${REGION}"`，Cloud Run 會自動把流量留在僅存的 revision 上；同樣未實測。

## 6. 本次本地演練結果

環境：macOS `arm64`（Apple Silicon）、Docker Desktop 29.6.1、`gcloud` **未安裝**（`command not found: gcloud`，未嘗試安裝，因為任務範圍是唯讀 `--version` 或不裝）。

### Build

```sh
docker build --platform=linux/amd64 -t context-rail-local:rehearsal -f Dockerfile .
```

- **結果：成功**，總耗時約 45 秒（多為 npm ci 與 go mod download 的網路/快取時間；`go build` 步驟本身約 22 秒）。
- **image 大小：4.15 MB**（`docker image inspect` 回報 `size=4151887`，架構 `amd64`/`linux`）——distroless static base + 靜態連結的 Go binary（`CGO_ENABLED=0`，`-trimpath -ldflags="-s -w"`，見 `Dockerfile:23`）+ 前端 dist（一個 JS bundle 約 360KB + CSS 16KB，`frontend/dist` 總計約幾百 KB）+ 兩個 fixture 目錄（504KB + 60KB）。沒有觸發任何 Dockerfile 或 `.dockerignore` 問題。
- 沒有跑 `docker build`（不加 `--platform`）比較 arm64/amd64 差異，因為沒有必要——Cloud Build 本來就在雲端 amd64 環境建置，本地是否用 `--platform` 只影響「這次演練的本地 tag」，不影響 10/6 實際部署路徑。

### Run + Smoke

```sh
docker run -d --name ctr-rehearsal --platform=linux/amd64 -p 18080:8080 \
  -e PORT=8080 \
  -e CONTEXT_RAIL_FIXTURE_ROOTS=/app/fixtures/order-operations-portal,/app/fixtures/support-insights \
  -e CONTEXT_RAIL_STATIC_DIR=/app/static \
  -e CONTEXT_RAIL_STATE_DIR=/tmp/context-rail-state \
  context-rail-local:rehearsal
```

四個環境變數其實與 Dockerfile 內建的 `ENV` 預設值完全相同（`Dockerfile:38-41`），這裡顯式帶入只是照任務指示做（真正需要覆寫的情境是換成 `CONTEXT_RAIL_FIXTURE_ROOTS=demo/order-operations-portal,examples/support-insights` 這種**相對路徑**，那是本機 `go run` 直接跑 binary 時用的寫法，不是容器內路徑；容器內固定是 `/app/fixtures/...`，两者不要混用）。

容器啟動 log：

```
ContextRail workspace listening on http://0.0.0.0:8080 (mode=container fixtures=[/app/fixtures/order-operations-portal /app/fixtures/support-insights] static=/app/static state=/tmp/context-rail-state advisor=rule-advisor read_back=none scenario=normal delay_ms=0)
```

curl 結果：

| 檢查 | 指令 | 結果 |
| --- | --- | --- |
| 健康檢查 | `curl http://127.0.0.1:18080/healthz` | `{"kind":"ContextRailHealth","status":"ok","mode":"container","surfaces":{...}}` |
| 前端靜態檔 | `curl -o /dev/null -w '%{http_code} %{content_type}' http://127.0.0.1:18080/` | `HTTP 200 content-type=text/html; charset=utf-8` |
| GET API | `curl http://127.0.0.1:18080/v1/projects` | 回傳兩個 fixture project（`order-operations-portal`、`support-insights`）的完整 registry snapshot |
| 唯讀邊界 | `curl -X POST http://127.0.0.1:18080/v1/projects` | `HTTP 405`（符合 `scripts/deploy-cloud-run.sh:21-22` 的 smoke 判斷邏輯） |

**這四項合起來就是 `scripts/deploy-cloud-run.sh` 裡 `smoke()` 函式（`scripts/deploy-cloud-run.sh:14-23`）在本地會做的完全相同的檢查**，只是這次打的是本地 port 18080 而不是 Cloud Run URL——這代表 smoke check 邏輯本身沒問題，10/6 部署到雲端後可以直接信任這支腳本。

跑完後：

```sh
docker stop ctr-rehearsal && docker rm ctr-rehearsal
```

**結果：容器已清除，沒有殘留**。

### 沒有跑的事（依任務範圍刻意留白）

- 沒有 `docker push`、沒有登入任何 registry、沒有跑任何 `gcloud` 寫入指令、沒有建立任何雲端資源。
- 沒有跑 `gcloud --version`（因為未安裝，`command not found`，已如實記錄，不用猜測版本）。
- 沒有驗證 Gemini 呼叫（本地沒有 `GEMINI_API_KEY`，容器 log 顯示 `advisor=rule-advisor`，這是預期行為，不是 bug）。
- 沒有跑 Playwright/E2E，只做了 curl 層級的手動 smoke（範圍內只要求健康檢查 + 一兩個 GET）。

## 7. 已知風險與未決項

1. **Secret Manager 路徑完全未實測**（第 2 節步驟 3-4 全部標「未實測」）——10/6 當天第一次跑務必保留每個指令的完整輸出，不要假設成功。
2. **`--min-instances=0`**（`cloudbuild.yaml:53`）代表 demo 展示中途若閒置過久，Cloud Run 可能把 instance 縮到零，`/tmp` 狀態（Change/Candidate/Release ledger）會全部遺失（`Dockerfile:34-37` 註解已明講）。10/6 展示前建議暫時改成 `--min-instances=1`（多花一點錢，換取展示期間不斷線），是否要改由 William 決定，本手冊只記錄風險，沒有代為修改 `cloudbuild.yaml`。
3. **Cloud Run 執行身份沿用專案預設 Compute SA 的權限範圍不明確**——如果 10/6 當天為了求快跳過第 2 節步驟 3（建專用 SA），直接用預設 SA 掛 secret，要留意預設 SA 在較舊專案上可能有 Editor 角色（過寬），這不是這次任務能決定的政策問題，只記錄。
4. **`go.mod:3` 寫 `go 1.22`，但 `Dockerfile:17` 與 `.github/workflows/ci.yml:21,49` 都用 Go 1.24**——目前不影響建置（新版 toolchain 可建置宣告較舊 module），只是版本宣告不一致，記錄但不算部署風險。
5. **Gemini 是否真的被呼叫到，目前只能靠回應內容裡的 `advisor_source` 或 fallback 訊息字串**（第 2 節步驟 6）——沒有專門的健康檢查端點或 log 欄位直接標示「這次請求用了哪個 advisor」；10/6 當天做完整測試後，若要長期化，值得考慮加一個明確的 debug 端點或 log 行（本次任務未實作，超出「本地建置演練」範圍）。
6. **`scripts/record-promotion.sh` 是給*受管理的 demo application*（`order-operations-portal` 之類）記錄部署用的，不是給 ContextRail 自己的部署記錄**——手冊第 4 節已經標明這個區分，避免 10/6 當天搞混兩支腳本的用途。
7. **這份手冊沒有動 `scripts/deploy-cloud-run.sh` / `cloudbuild.yaml` 本身**——所有新增步驟都是「額外跑的指令」，不是既有腳本的修改，這是刻意的決定（風險比較低，但代價是 10/6 當天要多跑幾個手動步驟，而不是一鍵完成）。如果 10/6 後確認這些步驟都跑得通，值得考慮把 secret/service-account 相關旗標併回 `cloudbuild.yaml`，那會是另一個 PR。
