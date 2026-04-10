Add-Type -AssemblyName System.Drawing

$root = Split-Path -Parent $PSScriptRoot
$buildDir = Join-Path $root 'build'
$iconsDir = Join-Path $buildDir 'icons'
$faviconDir = Join-Path $root 'frontend\public'

New-Item -ItemType Directory -Force -Path $buildDir | Out-Null
New-Item -ItemType Directory -Force -Path $iconsDir | Out-Null
New-Item -ItemType Directory -Force -Path $faviconDir | Out-Null

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

Write-Output "Generated build/icon.png, build/icon.ico, build/icons/* and frontend/public/favicon.png"
