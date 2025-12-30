# link-dl

Webページからファイルリンクを抽出して一括ダウンロードするCLIツール

## インストール

### 方法1: make install（推奨）

```bash
git clone https://github.com/IkumaTadokoro/link-dl
cd link-dl
make install
```

`/usr/local/bin` にインストールされ、どこからでも実行可能になります。

### 方法2: go install

```bash
go install github.com/IkumaTadokoro/link-dl@latest
```

`$GOPATH/bin`（通常 `~/go/bin`）にインストールされます。  
PATHに含まれていない場合は追加：

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### 方法3: 手動

```bash
git clone https://github.com/IkumaTadokoro/link-dl
cd link-dl
go build -o link-dl .
sudo mv link-dl /usr/local/bin/
```

### アンインストール

```bash
make uninstall
# または
sudo rm /usr/local/bin/link-dl
```

## 使い方

```bash
# PDF/Excelをダウンロード（デフォルト）
link-dl "https://www.wam.go.jp/gyoseiShiryou/detail?gno=20583"

# 拡張子を指定
link-dl "https://example.com/docs" --ext pdf,docx,zip

# 全ファイルリンクを対象
link-dl "https://example.com/files" --all

# URLパターンでフィルタ
link-dl "https://example.com/docs" --include "2024.*\.pdf"

# リスト表示のみ（ダウンロードしない）
link-dl "https://example.com/docs" --list

# 出力先と並列数を指定
link-dl "https://example.com/docs" --out ./my-folder --parallel 10
```

## オプション

| オプション | デフォルト | 説明 |
|------------|-----------|------|
| `--out` | `./downloads` | 出力ディレクトリ |
| `--parallel` | `5` | 並列ダウンロード数 |
| `--ext` | `pdf,xlsx,xls,xlsm` | 対象の拡張子（カンマ区切り） |
| `--all` | `false` | 全ファイルリンクを対象 |
| `--include` | - | URLフィルタ（正規表現） |
| `--list` | `false` | リスト表示のみ |
| `--ua` | Chrome UA | User-Agentヘッダー |

## 機能

- **リンクテキストをファイル名に**: `<a href="...">資料1</a>` → `資料1.pdf`
- **重複ファイル名の自動連番**: `file.pdf`, `file_2.pdf`, `file_3.pdf`
- **並列ダウンロード**: goroutineで高速化
- **柔軟なフィルタリング**: 拡張子指定、正規表現フィルタ

## 対応拡張子（--all モード）

pdf, doc, docx, xls, xlsx, xlsm, ppt, pptx, csv, txt, zip, rar, 7z, tar, gz, jpg, jpeg, png, gif, svg, mp3, mp4, wav, avi, mov

