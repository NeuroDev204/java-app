package javax.microedition.lcdui;

public class TextBox extends Screen {
    private String text;
    private int maxSize;
    private int constraints;

    public TextBox(String title, String text, int maxSize, int constraints) {
        super(title);
        this.text = text != null ? text : "";
        this.maxSize = maxSize;
        this.constraints = constraints;
    }

    public String getString() {
        return text;
    }

    public void setString(String text) {
        this.text = text != null ? text : "";
    }

    public int getChars(char[] data) {
        if (data == null) return 0;
        int len = Math.min(data.length, text.length());
        text.getChars(0, len, data, 0);
        return len;
    }

    public void setChars(char[] data, int offset, int length) {
        if (data != null && offset >= 0 && length >= 0 && offset + length <= data.length) {
            this.text = new String(data, offset, length);
        }
    }

    public void insert(String src, int position) {
        if (src == null) return;
        if (position < 0) position = 0;
        if (position > text.length()) position = text.length();
        text = text.substring(0, position) + src + text.substring(position);
    }

    public void insert(char[] data, int offset, int length, int position) {
        if (data == null) return;
        insert(new String(data, offset, length), position);
    }

    public void delete(int offset, int length) {
        if (offset < 0 || length <= 0 || offset >= text.length()) return;
        int end = Math.min(offset + length, text.length());
        text = text.substring(0, offset) + text.substring(end);
    }

    public int getMaxSize() {
        return maxSize;
    }

    public int setMaxSize(int maxSize) {
        this.maxSize = maxSize;
        return maxSize;
    }

    public int size() {
        return text.length();
    }

    public int getCaretPosition() {
        return text.length();
    }

    public int getConstraints() {
        return constraints;
    }

    public void setConstraints(int constraints) {
        this.constraints = constraints;
    }

    public void setInitialInputMode(String initialCharacterSubset) {}
}
