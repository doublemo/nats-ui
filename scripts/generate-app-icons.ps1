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

function Draw-Icon([int]$size, [string]$outPath) {
  $bitmap = New-Object System.Drawing.Bitmap $size, $size
  $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
  $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
  $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
  $graphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
  $graphics.Clear([System.Drawing.Color]::Transparent)

  $padding = [float]($size * 0.06)
  $radius = [float]($size * 0.22)
  $card = New-RoundedRectPath $padding $padding ($size - 2 * $padding) ($size - 2 * $padding) $radius

  $bgBrush = New-Object System.Drawing.Drawing2D.LinearGradientBrush(
    ([System.Drawing.PointF]::new(0, 0)),
    ([System.Drawing.PointF]::new($size, $size)),
    (New-Color 255 9 25 46),
    (New-Color 255 26 94 145)
  )
  $graphics.FillPath($bgBrush, $card)

  $glowBrushTop = New-Object System.Drawing.SolidBrush (New-Color 74 96 217 255)
  $graphics.FillEllipse($glowBrushTop, $size * 0.10, $size * 0.08, $size * 0.64, $size * 0.42)

  $glowBrushTopInner = New-Object System.Drawing.SolidBrush (New-Color 120 110 236 255)
  $graphics.FillEllipse($glowBrushTopInner, $size * 0.20, $size * 0.12, $size * 0.42, $size * 0.28)

  $glowBrushBottom = New-Object System.Drawing.SolidBrush (New-Color 72 38 196 255)
  $graphics.FillEllipse($glowBrushBottom, $size * 0.40, $size * 0.40, $size * 0.42, $size * 0.34)

  $glowBrushBottomInner = New-Object System.Drawing.SolidBrush (New-Color 108 72 222 255)
  $graphics.FillEllipse($glowBrushBottomInner, $size * 0.48, $size * 0.48, $size * 0.24, $size * 0.18)

  $borderPen = New-Object System.Drawing.Pen((New-Color 62 194 232 255), [float]($size * 0.006))
  $graphics.DrawPath($borderPen, $card)

  $orbitPen = New-Object System.Drawing.Pen((New-Color 58 151 216 255), [float]($size * 0.016))
  $orbitPen.StartCap = [System.Drawing.Drawing2D.LineCap]::Round
  $orbitPen.EndCap = [System.Drawing.Drawing2D.LineCap]::Round
  $graphics.DrawArc($orbitPen, $size * 0.22, $size * 0.14, $size * 0.5, $size * 0.34, 208, 108)
  $graphics.DrawArc($orbitPen, $size * 0.34, $size * 0.44, $size * 0.42, $size * 0.28, 26, 108)

  $shadowPen = New-Object System.Drawing.Pen((New-Color 75 5 14 28), [float]($size * 0.13))
  $shadowPen.StartCap = [System.Drawing.Drawing2D.LineCap]::Round
  $shadowPen.EndCap = [System.Drawing.Drawing2D.LineCap]::Round
  $shadowPen.LineJoin = [System.Drawing.Drawing2D.LineJoin]::Round

  $strokeBrush = New-Object System.Drawing.Drawing2D.LinearGradientBrush(
    ([System.Drawing.PointF]::new($size * 0.22, $size * 0.18)),
    ([System.Drawing.PointF]::new($size * 0.78, $size * 0.82)),
    (New-Color 255 240 248 255),
    (New-Color 255 101 235 255)
  )
  $strokePen = New-Object System.Drawing.Pen($strokeBrush, [float]($size * 0.11))
  $strokePen.StartCap = [System.Drawing.Drawing2D.LineCap]::Round
  $strokePen.EndCap = [System.Drawing.Drawing2D.LineCap]::Round
  $strokePen.LineJoin = [System.Drawing.Drawing2D.LineJoin]::Round

  $leftTop = [System.Drawing.PointF]::new($size * 0.30, $size * 0.25)
  $leftBottom = [System.Drawing.PointF]::new($size * 0.30, $size * 0.74)
  $rightTop = [System.Drawing.PointF]::new($size * 0.70, $size * 0.25)
  $rightBottom = [System.Drawing.PointF]::new($size * 0.70, $size * 0.74)

  $graphics.DrawLine($shadowPen, $leftBottom.X + $size * 0.018, $leftBottom.Y + $size * 0.018, $leftTop.X + $size * 0.018, $leftTop.Y + $size * 0.018)
  $graphics.DrawLine($shadowPen, $leftTop.X + $size * 0.018, $leftTop.Y + $size * 0.018, $rightBottom.X + $size * 0.018, $rightBottom.Y + $size * 0.018)
  $graphics.DrawLine($shadowPen, $rightBottom.X + $size * 0.018, $rightBottom.Y + $size * 0.018, $rightTop.X + $size * 0.018, $rightTop.Y + $size * 0.018)

  $graphics.DrawLine($strokePen, $leftBottom, $leftTop)
  $graphics.DrawLine($strokePen, $leftTop, $rightBottom)
  $graphics.DrawLine($strokePen, $rightBottom, $rightTop)

  $nodeBrush = New-Object System.Drawing.SolidBrush (New-Color 255 196 249 255)
  $nodeGlow = New-Object System.Drawing.SolidBrush (New-Color 90 90 225 255)
  $nodeRadius = [float]($size * 0.055)
  $nodePositions = @(
    [System.Drawing.PointF]::new($size * 0.24, $size * 0.20),
    [System.Drawing.PointF]::new($size * 0.50, $size * 0.49),
    [System.Drawing.PointF]::new($size * 0.76, $size * 0.79)
  )

  foreach ($point in $nodePositions) {
    $graphics.FillEllipse($nodeGlow, $point.X - $nodeRadius * 1.28, $point.Y - $nodeRadius * 1.28, $nodeRadius * 2.56, $nodeRadius * 2.56)
    $graphics.FillEllipse($nodeBrush, $point.X - $nodeRadius, $point.Y - $nodeRadius, $nodeRadius * 2, $nodeRadius * 2)
  }

  $bitmap.Save($outPath, [System.Drawing.Imaging.ImageFormat]::Png)

  $nodeBrush.Dispose()
  $nodeGlow.Dispose()
  $strokePen.Dispose()
  $strokeBrush.Dispose()
  $shadowPen.Dispose()
  $orbitPen.Dispose()
  $borderPen.Dispose()
  $glowBrushBottomInner.Dispose()
  $glowBrushBottom.Dispose()
  $glowBrushTopInner.Dispose()
  $glowBrushTop.Dispose()
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
