// ANSI escape sequence to HTML converter
export function convertAnsiToHtml(text) {
    // ANSI color codes to CSS classes/styles
    const ansiColors = {
        // Standard colors (30-37, 90-97)
        30: '#000000', // black
        31: '#cd0000', // red
        32: '#00cd00', // green
        33: '#cdcd00', // yellow
        34: '#0000ee', // blue
        35: '#cd00cd', // magenta
        36: '#00cdcd', // cyan
        37: '#e5e5e5', // white

        // Bright colors (90-97)
        90: '#7f7f7f', // bright black (gray)
        91: '#ff0000', // bright red
        92: '#00ff00', // bright green
        93: '#ffff00', // bright yellow
        94: '#5c5cff', // bright blue
        95: '#ff00ff', // bright magenta
        96: '#00ffff', // bright cyan
        97: '#ffffff', // bright white
    };

    const ansiBgColors = {
        // Background colors (40-47, 100-107)
        40: '#000000', // black bg
        41: '#cd0000', // red bg
        42: '#00cd00', // green bg
        43: '#cdcd00', // yellow bg
        44: '#0000ee', // blue bg
        45: '#cd00cd', // magenta bg
        46: '#00cdcd', // cyan bg
        47: '#e5e5e5', // white bg

        // Bright background colors (100-107)
        100: '#7f7f7f', // bright black bg
        101: '#ff0000', // bright red bg
        102: '#00ff00', // bright green bg
        103: '#ffff00', // bright yellow bg
        104: '#5c5cff', // bright blue bg
        105: '#ff00ff', // bright magenta bg
        106: '#00ffff', // bright cyan bg
        107: '#ffffff', // bright white bg
    };

    // State tracking
    let currentColor = null;
    let currentBgColor = null;
    let bold = false;
    let italic = false;
    let underline = false;

    // Remove cursor positioning and screen control sequences but keep colors
    let processedText = text
        // Remove cursor positioning (ESC[row;colH or ESC[row;colf)
        .replace(/\x1b\[[0-9]+;[0-9]+[Hf]/g, '')
        // Remove scrolling region (ESC[top;bottomr)
        .replace(/\x1b\[[0-9]+;[0-9]+r/g, '')
        // Remove single number scrolling region (ESC[numberr)
        .replace(/\x1b\[[0-9]+r/g, '')
        // Remove cursor movement
        .replace(/\x1b\[[ABCD]/g, '')
        // Remove screen clearing
        .replace(/\x1b\[2J/g, '')
        // Remove line clearing
        .replace(/\x1b\[[0-2]?K/g, '')
        // Remove cursor save/restore
        .replace(/\x1b[78]/g, '')
        // Remove cursor position queries
        .replace(/\x1b\[[0-9]*;[0-9]*R/g, '')
        // Remove device status report queries
        .replace(/\x1b\[[0-9]*n/g, '')
        // Remove other positioning sequences
        .replace(/\x1b\[[0-9]*[JK]/g, '')
        // Remove cursor visibility
        .replace(/\x1b\[\?25[lh]/g, '')
        // Remove application keypad mode
        .replace(/\x1b\[[\?=][0-9]*[lh]/g, '');

    // Process ANSI color codes
    const result = processedText.replace(/\x1b\[([0-9;]*)m/g, (match, codes) => {
        if (!codes) codes = '0'; // Default reset

        const codeList = codes.split(';').map(code => parseInt(code) || 0);
        let styles = [];

        for (const code of codeList) {
            switch (code) {
                case 0: // Reset all
                    currentColor = null;
                    currentBgColor = null;
                    bold = false;
                    italic = false;
                    underline = false;
                    return '</span><span class="ansi-text">';

                case 1: // Bold
                    bold = true;
                    break;
                case 3: // Italic
                    italic = true;
                    break;
                case 4: // Underline
                    underline = true;
                    break;
                case 22: // Bold off
                    bold = false;
                    break;
                case 23: // Italic off
                    italic = false;
                    break;
                case 24: // Underline off
                    underline = false;
                    break;

                // Foreground colors
                default:
                    if (ansiColors[code]) {
                        currentColor = ansiColors[code];
                    } else if (ansiBgColors[code]) {
                        currentBgColor = ansiBgColors[code];
                    }
                    break;
            }
        }

        // Build style string
        if (currentColor) styles.push(`color: ${currentColor}`);
        if (currentBgColor) styles.push(`background-color: ${currentBgColor}`);
        if (bold) styles.push('font-weight: bold');
        if (italic) styles.push('font-style: italic');
        if (underline) styles.push('text-decoration: underline');

        const styleStr = styles.length > 0 ? ` style="${styles.join('; ')}"` : '';
        return `</span><span class="ansi-text"${styleStr}>`;
    });

    // Wrap the entire result and clean up
    const wrapped = `<span class="ansi-text">${result}</span>`;

    // Clean up empty spans
    return wrapped
        .replace(/<span class="ansi-text"><\/span>/g, '')
        .replace(/<span class="ansi-text">\s*<\/span>/g, '');
}

// React component for rendering ANSI text
export function AnsiText({ children }) {
    const htmlContent = convertAnsiToHtml(children || '');

    return (
        <span
            className="ansi-container"
            dangerouslySetInnerHTML={{ __html: htmlContent }}
        />
    );
}