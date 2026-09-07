package javax.microedition.lcdui;

import java.util.Vector;

public abstract class Displayable {
    protected Vector commands = new Vector();
    protected CommandListener listener;
    protected String title;

    public Displayable() {}
    public Displayable(String title) { this.title = title; }

    public String getTitle() { return title; }
    public void setTitle(String title) { this.title = title; }

    public int getWidth() { return 240; }
    public int getHeight() { return 320; }
    public boolean isShown() { return true; }
    
    public void addCommand(Command cmd) {
        if (!commands.contains(cmd)) commands.addElement(cmd);
    }
    public void removeCommand(Command cmd) {
        commands.removeElement(cmd);
    }
    public void setCommandListener(CommandListener l) {
        this.listener = l;
    }
    
    public void fireCommandAction(Command c) {
        if (listener != null && c != null) {
            listener.commandAction(c, this);
        }
    }
    
    public Command getOkCommand() {
        if (commands == null) return null;
        for (int i = 0; i < commands.size(); i++) {
            Command c = (Command)commands.elementAt(i);
            if (c.getCommandType() == Command.OK || c.getCommandType() == Command.ITEM) {
                return c;
            }
        }
        if (commands.size() > 0) return (Command)commands.elementAt(0);
        return null;
    }
}
