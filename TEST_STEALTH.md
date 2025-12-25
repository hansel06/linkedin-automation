# Stealth Implementation Test

This document describes how to test the Phase 7 stealth implementation.

## Quick Test

Run the test program:

```bash
go run ./cmd/test-stealth
```

Or build and run:

```bash
go build -o test-stealth.exe ./cmd/test-stealth
./test-stealth.exe
```

## What the Test Does

The test program verifies all stealth features:

1. ✅ **RandomDelay()** - Random delays between actions
2. ✅ **WaitHumanized()** - Thinking time before actions
3. ✅ **MoveMouse()** - Bezier curve mouse movement (watch for curved path!)
4. ✅ **Scroll()** - Variable speed scrolling with easing
5. ✅ **RandomScroll()** - Random scroll patterns
6. ✅ **Type()** - Human-like typing with typos and reading pauses
7. ✅ **Hover()** - Mouse wandering before settling on element
8. ✅ **Click()** - Human-like clicking with Bezier movement

## Expected Behavior

When you run the test:

1. **Browser opens** (visible window)
2. **Navigates to example.com**
3. **Mouse movement test**: You should see the mouse move in a **curved path** (not straight line) from one corner to another
4. **Scrolling test**: Page scrolls with **variable speed** (slow start, fast middle, slow end)
5. **Typing test**: Navigates to Google, types in search box with **human-like timing** (you may see typos being corrected)
6. **Hover test**: Mouse **wanders slightly** before settling on a link
7. **Click test**: Mouse moves in a **curve** before clicking a link

## Visual Verification

**Key things to watch for:**

- ✅ Mouse moves in **curved paths**, not straight lines
- ✅ Mouse movement has **overshoot** (goes past target, then corrects)
- ✅ Scrolling has **variable speed** (not constant)
- ✅ Typing has **pauses** between characters (not instant)
- ✅ Mouse **wanders** before hovering (not direct movement)
- ✅ Click has **thinking time** before clicking

## Troubleshooting

### Mouse doesn't move
- Check if browser window is visible
- Check console for errors
- Ensure page has loaded completely

### Typing doesn't work
- Google may have changed their page structure
- Test will skip typing if search box not found (this is okay)

### Scrolling doesn't work
- Check if page is scrollable
- Some pages may not scroll if content fits in viewport

## Test Results

If you see:
- ✅ Curved mouse movements
- ✅ Variable speed scrolling
- ✅ Human-like typing behavior
- ✅ Mouse wandering before hover
- ✅ Natural click behavior

Then **Phase 7 stealth implementation is working correctly!**


