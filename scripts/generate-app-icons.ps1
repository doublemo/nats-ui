$ErrorActionPreference = 'Stop'

Add-Type -AssemblyName System.Drawing

$root = Split-Path -Parent $PSScriptRoot
$buildDir = Join-Path $root 'build'
$iconsDir = Join-Path $buildDir 'icons'
$faviconDir = Join-Path $root 'frontend\public'
$installerDir = Join-Path $buildDir 'installer'

New-Item -ItemType Directory -Force -Path $buildDir | Out-Null
New-Item -ItemType Directory -Force -Path $iconsDir | Out-Null
New-Item -ItemType Directory -Force -Path $faviconDir | Out-Null
New-Item -ItemType Directory -Force -Path $installerDir | Out-Null

function New-Color([int]$a, [int]$r, [int]$g, [int]$b) {
  return [System.Drawing.Color]::FromArgb($a, $r, $g, $b)
}

function New-RoundedRectPath([float]$x, [float]$y, [float]$width, [float]$height, [float]$radius) {
  $path = New-Object System.Drawing.Drawing2D.GraphicsPath
  $diameter = $radius * 2

  $path.AddArc($x, $y, $diameter, $diameter, 180, 90)
  $path.AddArc($x + $width - $diameter, $y, $diameter, $diameter, 270, 90)
  $path.AddArc($x + $width - $diameter, $y + $height - $diameter, $diameter, $diameter, 0, 90)
  $path.AddArc($x, $y + $height - $diameter, $diameter, $diameter, 90, 90)
  $path.CloseFigure()

  return $path
}

# A publisher fans out to three subscribers. Supersampling keeps small icons crisp.
function Draw-Icon([int]$size, [string]$outPath) {
  $renderSize = [Math]::Max(256, $size * 4)
  $bitmap = New-Object System.Drawing.Bitmap $renderSize, $renderSize
  $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
  $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
  $graphics.Clear([System.Drawing.Color]::Transparent)
  $graphics.ScaleTransform(($renderSize / 100.0), ($renderSize / 100.0))
  $card = New-RoundedRectPath 4 4 92 92 23
  $bgBrush = New-Object System.Drawing.Drawing2D.LinearGradientBrush(
    ([System.Drawing.PointF]::new(10, 4)),
    ([System.Drawing.PointF]::new(88, 96)),
    (New-Color 255 111 87 242),
    (New-Color 255 57 42 154)
  )
  $graphics.FillPath($bgBrush, $card)
  $routePen = New-Object System.Drawing.Pen((New-Color 255 247 248 255), 8)
  $routePen.StartCap = [System.Drawing.Drawing2D.LineCap]::Round
  $routePen.EndCap = [System.Drawing.Drawing2D.LineCap]::Round
  $routePen.LineJoin = [System.Drawing.Drawing2D.LineJoin]::Round
  $route = New-Object System.Drawing.Drawing2D.GraphicsPath
  $route.AddLine(26, 50, 40, 50)
  $route.AddBezier(40, 50, 52, 50, 48, 28, 63, 28)
  $route.AddLine(63, 28, 73, 28)
  $graphics.DrawPath($routePen, $route)
  $route.Reset()
  $route.AddLine(26, 50, 73, 50)
  $graphics.DrawPath($routePen, $route)
  $route.Reset()
  $route.AddLine(26, 50, 40, 50)
  $route.AddBezier(40, 50, 52, 50, 48, 72, 63, 72)
  $route.AddLine(63, 72, 73, 72)
  $graphics.DrawPath($routePen, $route)
  $sourceBrush = New-Object System.Drawing.SolidBrush (New-Color 255 247 248 255)
  $nodeBrush = New-Object System.Drawing.SolidBrush (New-Color 255 125 249 207)
  $graphics.FillEllipse($sourceBrush, 16, 40, 20, 20)
  foreach ($y in @(28, 50, 72)) {
    $graphics.FillEllipse($nodeBrush, 66, ($y - 7), 14, 14)
  }
  $output = New-Object System.Drawing.Bitmap $size, $size
  $outputGraphics = [System.Drawing.Graphics]::FromImage($output)
  $outputGraphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
  $outputGraphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
  $outputGraphics.DrawImage($bitmap, 0, 0, $size, $size)
  $output.Save($outPath, [System.Drawing.Imaging.ImageFormat]::Png)
  $outputGraphics.Dispose()
  $output.Dispose()
  $nodeBrush.Dispose()
  $sourceBrush.Dispose()
  $route.Dispose()
  $routePen.Dispose()
  $bgBrush.Dispose()
  $card.Dispose()
  $graphics.Dispose()
  $bitmap.Dispose()
}
function Draw-InstallerPanel([int]$width, [int]$height, [string]$outPath) {
  $bitmap = New-Object System.Drawing.Bitmap $width, $height
  $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
  $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
  $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
  $graphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality

  $background = New-Object System.Drawing.Drawing2D.LinearGradientBrush(
    ([System.Drawing.PointF]::new(0, 0)),
    ([System.Drawing.PointF]::new($width, $height)),
    (New-Color 255 7 21 39),
    (New-Color 255 15 67 112)
  )
  $graphics.FillRectangle($background, 0, 0, $width, $height)

  $glowA = New-Object System.Drawing.SolidBrush (New-Color 60 84 205 255)
  $glowB = New-Object System.Drawing.SolidBrush (New-Color 80 42 189 255)
  $glowC = New-Object System.Drawing.SolidBrush (New-Color 96 120 233 255)
  $graphics.FillEllipse($glowA, -$width * 0.10, $height * 0.02, $width * 0.92, $height * 0.40)
  $graphics.FillEllipse($glowB, $width * 0.24, $height * 0.46, $width * 0.70, $height * 0.34)
  $graphics.FillEllipse($glowC, $width * 0.12, $height * 0.18, $width * 0.52, $height * 0.18)

  $iconPath = Join-Path $buildDir 'icon.png'
  $iconSize = [int]([Math]::Min($width * 0.52, $height * 0.28))
  $iconX = [int](($width - $iconSize) / 2)
  $iconY = [int]($height * 0.11)
  if (Test-Path $iconPath) {
    $iconImage = [System.Drawing.Image]::FromFile($iconPath)
    $graphics.DrawImage($iconImage, $iconX, $iconY, $iconSize, $iconSize)
    $iconImage.Dispose()
  }

  $lineBrush = New-Object System.Drawing.SolidBrush (New-Color 255 235 247 255)
  $subBrush = New-Object System.Drawing.SolidBrush (New-Color 208 214 232 255)
  $accentBrush = New-Object System.Drawing.SolidBrush (New-Color 255 125 230 255)

  $titleFont = New-Object System.Drawing.Font('Segoe UI', [float]($height * 0.050), [System.Drawing.FontStyle]::Bold, [System.Drawing.GraphicsUnit]::Pixel)
  $subFont = New-Object System.Drawing.Font('Segoe UI', [float]($height * 0.027), [System.Drawing.FontStyle]::Regular, [System.Drawing.GraphicsUnit]::Pixel)
  $smallFont = New-Object System.Drawing.Font('Segoe UI', [float]($height * 0.024), [System.Drawing.FontStyle]::Regular, [System.Drawing.GraphicsUnit]::Pixel)

  $textX = [float]($width * 0.13)
  $titleY = [float]($height * 0.51)
  $graphics.DrawString('NATS UI', $titleFont, $lineBrush, $textX, $titleY)
  $graphics.DrawString('Desktop Installer', $subFont, $accentBrush, $textX, $titleY + $height * 0.12)
  $graphics.DrawString('Manage NATS, JetStream and KV from one workspace.', $smallFont, $subBrush, $textX, $titleY + $height * 0.22)
  $graphics.DrawString('MIT Licensed', $smallFont, $subBrush, $textX, $height * 0.90)

  $bitmap.Save($outPath, [System.Drawing.Imaging.ImageFormat]::Bmp)

  $smallFont.Dispose()
  $subFont.Dispose()
  $titleFont.Dispose()
  $accentBrush.Dispose()
  $subBrush.Dispose()
  $lineBrush.Dispose()
  $glowC.Dispose()
  $glowB.Dispose()
  $glowA.Dispose()
  $background.Dispose()
  $graphics.Dispose()
  $bitmap.Dispose()
}

function Draw-DmgBackground([int]$width, [int]$height, [string]$outPath) {
  $bitmap = New-Object System.Drawing.Bitmap $width, $height
  $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
  $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
  $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
  $graphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality

  $background = New-Object System.Drawing.Drawing2D.LinearGradientBrush(
    ([System.Drawing.PointF]::new(0, 0)),
    ([System.Drawing.PointF]::new($width, $height)),
    (New-Color 255 239 244 250),
    (New-Color 255 226 236 246)
  )
  $graphics.FillRectangle($background, 0, 0, $width, $height)

  $orbA = New-Object System.Drawing.SolidBrush (New-Color 70 115 206 255)
  $orbB = New-Object System.Drawing.SolidBrush (New-Color 62 67 192 255)
  $orbC = New-Object System.Drawing.SolidBrush (New-Color 105 119 231 255)
  $graphics.FillEllipse($orbA, -$width * 0.05, -$height * 0.12, $width * 0.42, $height * 0.58)
  $graphics.FillEllipse($orbB, $width * 0.66, $height * 0.06, $width * 0.30, $height * 0.42)
  $graphics.FillEllipse($orbC, $width * 0.32, $height * 0.58, $width * 0.36, $height * 0.28)

  $cardPath = New-RoundedRectPath ($width * 0.05) ($height * 0.08) ($width * 0.48) ($height * 0.76) ($height * 0.08)
  $cardBrush = New-Object System.Drawing.SolidBrush (New-Color 210 9 25 46)
  $graphics.FillPath($cardBrush, $cardPath)

  $iconImage = [System.Drawing.Image]::FromFile((Join-Path $buildDir 'icon.png'))
  $iconSize = [int]($height * 0.22)
  $graphics.DrawImage($iconImage, [int]($width * 0.12), [int]($height * 0.16), $iconSize, $iconSize)
  $iconImage.Dispose()

  $titleFont = New-Object System.Drawing.Font('Segoe UI', [float]($height * 0.076), [System.Drawing.FontStyle]::Bold, [System.Drawing.GraphicsUnit]::Pixel)
  $bodyFont = New-Object System.Drawing.Font('Segoe UI', [float]($height * 0.036), [System.Drawing.FontStyle]::Regular, [System.Drawing.GraphicsUnit]::Pixel)
  $hintFont = New-Object System.Drawing.Font('Segoe UI', [float]($height * 0.030), [System.Drawing.FontStyle]::Regular, [System.Drawing.GraphicsUnit]::Pixel)

  $whiteBrush = New-Object System.Drawing.SolidBrush (New-Color 255 244 249 255)
  $mutedBrush = New-Object System.Drawing.SolidBrush (New-Color 215 219 232 245)
  $accentBrush = New-Object System.Drawing.SolidBrush (New-Color 255 120 232 255)

  $graphics.DrawString('NATS UI', $titleFont, $whiteBrush, $width * 0.12, $height * 0.46)
  $graphics.DrawString('Desktop Workspace for NATS, JetStream and KV', $bodyFont, $accentBrush, $width * 0.12, $height * 0.60)
  $graphics.DrawString('Drag the app icon into Applications to install.', $hintFont, $mutedBrush, $width * 0.12, $height * 0.72)

  $guidePen = New-Object System.Drawing.Pen((New-Color 80 255 255 255), 3)
  $guidePen.DashStyle = [System.Drawing.Drawing2D.DashStyle]::Dash
  $graphics.DrawLine($guidePen, $width * 0.60, $height * 0.52, $width * 0.82, $height * 0.52)

  $bitmap.Save($outPath, [System.Drawing.Imaging.ImageFormat]::Png)

  $guidePen.Dispose()
  $accentBrush.Dispose()
  $mutedBrush.Dispose()
  $whiteBrush.Dispose()
  $hintFont.Dispose()
  $bodyFont.Dispose()
  $titleFont.Dispose()
  $cardBrush.Dispose()
  $cardPath.Dispose()
  $orbC.Dispose()
  $orbB.Dispose()
  $orbA.Dispose()
  $background.Dispose()
  $graphics.Dispose()
  $bitmap.Dispose()
}

function New-Ico([string]$outPath, [int[]]$sizes) {
  $frames = @()
  foreach ($size in $sizes) {
    $path = Join-Path $iconsDir "$($size)x$($size).png"
    $frames += [PSCustomObject]@{
      Size = $size
      Bytes = [System.IO.File]::ReadAllBytes($path)
    }
  }

  $stream = [System.IO.File]::Create($outPath)
  $writer = New-Object System.IO.BinaryWriter($stream)

  $writer.Write([UInt16]0)
  $writer.Write([UInt16]1)
  $writer.Write([UInt16]$frames.Count)

  $offset = 6 + ($frames.Count * 16)

  foreach ($frame in $frames) {
    $dimension = if ($frame.Size -ge 256) { 0 } else { [byte]$frame.Size }
    $writer.Write([byte]$dimension)
    $writer.Write([byte]$dimension)
    $writer.Write([byte]0)
    $writer.Write([byte]0)
    $writer.Write([UInt16]1)
    $writer.Write([UInt16]32)
    $writer.Write([UInt32]$frame.Bytes.Length)
    $writer.Write([UInt32]$offset)
    $offset += $frame.Bytes.Length
  }

  foreach ($frame in $frames) {
    $writer.Write($frame.Bytes)
  }

  $writer.Dispose()
  $stream.Dispose()
}

$pngSizes = @(16, 24, 32, 48, 64, 128, 256, 512, 1024)
foreach ($size in $pngSizes) {
  Draw-Icon -size $size -outPath (Join-Path $iconsDir "$($size)x$($size).png")
}

Copy-Item (Join-Path $iconsDir '512x512.png') (Join-Path $buildDir 'icon.png') -Force
Copy-Item (Join-Path $iconsDir '256x256.png') (Join-Path $faviconDir 'favicon.png') -Force
New-Ico -outPath (Join-Path $buildDir 'icon.ico') -sizes @(16, 24, 32, 48, 64, 128, 256)
Copy-Item (Join-Path $buildDir 'icon.ico') (Join-Path $buildDir 'installerIcon.ico') -Force
Copy-Item (Join-Path $buildDir 'icon.ico') (Join-Path $buildDir 'uninstallerIcon.ico') -Force
Copy-Item (Join-Path $buildDir 'icon.ico') (Join-Path $buildDir 'installerHeaderIcon.ico') -Force
Draw-InstallerPanel -width 164 -height 314 -outPath (Join-Path $installerDir 'installerSidebar.bmp')
Draw-InstallerPanel -width 164 -height 314 -outPath (Join-Path $installerDir 'uninstallerSidebar.bmp')
Draw-DmgBackground -width 1080 -height 760 -outPath (Join-Path $installerDir 'background.png')
if (Test-Path (Join-Path $root 'LICENSE')) {
  Copy-Item (Join-Path $root 'LICENSE') (Join-Path $buildDir 'license.txt') -Force
}

Write-Output "Generated build/icon.png, build/icon.ico, build/installer/*, build/icons/* and frontend/public/favicon.png"
