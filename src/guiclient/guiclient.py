#! /usr/bin/env python2
# -*- coding: UTF-8 -*-
#################################################################################
#     File Name           :     guiclient.py
#     Created By          :     YIMMON, yimmon.zhuang@gmail.com
#     Creation Date       :     [2014-04-09 21:30]
#     Last Modified       :     [2014-08-15 22:52]
#     Description         :
#################################################################################

import time
import math
import socket
import simplejson
import glib
import gtk
import pango
import threading


class DAB(gtk.Window):
    def __init__(self):
        super(DAB, self).__init__()

        self.myname, self.version = 'Qdab', '0.1'
        self.set_title(self.myname)
        self.set_resizable(False)
        self.set_position(gtk.WIN_POS_CENTER)
        self.GridSize = 60
        self.thinking = False

        agr = gtk.AccelGroup()
        self.add_accel_group(agr)

        self.mb = gtk.MenuBar()
        self.fileitem = gtk.MenuItem('File')
        self.filemenu = gtk.Menu()
        self.fileitem.set_submenu(self.filemenu)

        self.newitem = gtk.MenuItem('New')
        key, mod = gtk.accelerator_parse('<Control>N')
        self.newitem.add_accelerator('activate', agr, key, mod, gtk.ACCEL_VISIBLE)
        self.newitem.connect('activate', self.on_new_game)
        self.filemenu.append(self.newitem)
        self.humanitem = gtk.CheckMenuItem('Human first')
        self.humanitem.set_sensitive(False)
        self.humanitem.set_active(True)
        self.humanitem.connect('activate', self.on_human_first)
        self.filemenu.append(self.humanitem)
        self.robotitem = gtk.CheckMenuItem('Robot first')
        self.robotitem.set_active(False)
        self.robotitem.connect('activate', self.on_robot_first)
        self.filemenu.append(self.robotitem)
        self.filemenu.append(gtk.SeparatorMenuItem())
        self.undoitem = gtk.MenuItem('Undo')
        self.undoitem.set_sensitive(False)
        key, mod = gtk.accelerator_parse('<Control>X')
        self.undoitem.add_accelerator('activate', agr, key, mod, gtk.ACCEL_VISIBLE)
        self.undoitem.connect('activate', self.on_undo)
        self.filemenu.append(self.undoitem)
        self.redoitem = gtk.MenuItem('Redo')
        self.redoitem.set_sensitive(False)
        key, mod = gtk.accelerator_parse('<Control>Z')
        self.redoitem.add_accelerator('activate', agr, key, mod, gtk.ACCEL_VISIBLE)
        self.redoitem.connect('activate', self.on_redo)
        self.filemenu.append(self.redoitem)
        self.filemenu.append(gtk.SeparatorMenuItem())
        self.exititem = gtk.MenuItem('Exit')
        key, mod = gtk.accelerator_parse('<Control>Q')
        self.exititem.add_accelerator('activate', agr, key, mod, gtk.ACCEL_VISIBLE)
        self.exititem.connect('activate', gtk.main_quit)
        self.filemenu.append(self.exititem)
        self.mb.append(self.fileitem)

        self.aboutitem = gtk.MenuItem('About')
        self.aboutitem.connect('activate', self.on_about_activate)
        self.mb.append(self.aboutitem)

        self.darea = gtk.DrawingArea()
        self.darea.set_size_request(self.GridSize * 7, self.GridSize * 8)
        self.darea.modify_bg(gtk.STATE_NORMAL, gtk.gdk.Color(
            240 * 257, 248 * 257, 255 * 257))

        self.statusbar = gtk.Statusbar()
        self.statusbar.push(1, ' 0m00s Move(s): 0')
        align = gtk.Alignment(0.9, 0, 0, 1)
        self.scorelabel = gtk.Label('Human: 0, Robot: 0')
        align.add(self.scorelabel)
        self.statusbar.add(align)

        self.vbox = gtk.VBox(False, 2)
        self.vbox.pack_start(self.mb, False, False, 0)
        self.vbox.pack_start(self.darea, False, False, 0)
        self.vbox.pack_start(self.statusbar, False, False, 0)
        self.add(self.vbox)

        self.adjust_timeout, self.timeout_offset = False, 1.0 / 6.0
        self.t_x1, self.t_x2, self.t_y = self.GridSize * 2, self.GridSize * 5, int(self.GridSize * 7)

        self.connect('destroy', gtk.main_quit)
        self.darea.connect('expose-event', self.expose)
        self.darea.add_events(gtk.gdk.ALL_EVENTS_MASK)
        self.darea.connect('button-press-event', self.on_darea_button_press)
        self.darea.connect('button-release-event', self.on_darea_button_release)
        self.darea.connect('motion-notify-event', self.on_darea_motion_notify)
        self.darea.connect('leave-notify-event', self.on_darea_leave_notify)
        self.init_board()

        self.show_all()

    def init_board(self, first=0):
        self.degree = [[4] * 5 for i in range(5)]
        self.belong = [[-1] * 5 for i in range(5)]
        self.hexist = [[0] * 5 for i in range(6)]
        self.vexist = [[0] * 6 for i in range(5)]
        self.human, self.robot = 0, 0
        self.first, self.who = first, first
        self.moves, self.record = 0, []
        self.turn = 0
        self.cursor = (-1, -1, -1)
        self.begtime = time.time()
        self.update_time_elapse()
        self.queue_draw()
        if self.who != 0:
            Robot(self).start()

    def expose(self, widget, event):
        self.undoitem.set_sensitive(self.moves > 0)
        self.redoitem.set_sensitive(self.moves < len(self.record))
        self.scorelabel.set_text(' Human: %d, Robot: %d' % (self.human, self.robot))
        self.draw_board()

    def update_time_elapse(self):
        self.statusbar.remove_all(1)
        if self.thinking:
            tips = 'thinking...'
        else:
            tips = ' Turn(s): %d' % self.turn
        self.statusbar.push(
            1, ' %dm%02ds %s' % (int(time.time() - self.begtime) / 60,
                                 int(time.time() - self.begtime) % 60, tips))
        if self.human + self.robot < 25:
            glib.timeout_add(1000, self.update_time_elapse)
        self.queue_draw()

    def on_new_game(self, widget):
        if self.thinking:
            return
        self.init_board(self.first)

    def on_human_first(self, widget):
        if self.thinking:
            return
        if self.first == 1:
            self.robotitem.set_active(False)
            self.robotitem.set_sensitive(True)
            self.humanitem.set_sensitive(False)
            self.first = 0
            self.init_board(0)

    def on_robot_first(self, widget):
        if self.thinking:
            return
        if self.first == 0:
            self.humanitem.set_active(False)
            self.humanitem.set_sensitive(True)
            self.robotitem.set_sensitive(False)
            self.first = 1
            self.init_board(1)

    def on_undo(self, widget):
        if self.thinking:
            return
        for i in range(self.moves)[::-1]:
            if self.record[i][3] == 0:
                self.moves = i
                break
        self.degree = [[4] * 5 for i in range(5)]
        self.belong = [[-1] * 5 for i in range(5)]
        self.hexist = [[0] * 5 for i in range(6)]
        self.vexist = [[0] * 6 for i in range(5)]
        self.human, self.robot = 0, 0
        self.who = self.first
        self.turn = 0
        for move in self.record[:self.moves]:
            self.move(move, False)
        self.queue_draw()

    def on_redo(self, widget):
        if self.thinking:
            return
        cnt = 0
        for move in self.record[self.moves:]:
            if move[3] == 0:
                cnt += 1
            if cnt > 1:
                break
            self.move(move, False)
            self.moves += 1
        self.queue_draw()

    def on_darea_button_press(self, widget, event):
        x = self.t_x1+(self.t_x2-self.t_x1)*self.timeout_offset
        if math.sqrt((event.x-x) ** 2 + (event.y-self.t_y) ** 2) < 10:
            self.adjust_timeout = True

    def on_darea_button_release(self, widget, event):
        if self.adjust_timeout:
            self.adjust_timeout = False
            return
        if self.thinking:
            return
        if self.who == 0:  # human's turn
            move = self.xy2move(event.x, event.y, self.who)
            if move[0] < 0:
                return
            self.move(move)
            self.cursor = (-1, -1, -1)
            self.queue_draw()
            if self.who != 0:
                Robot(self).start()

    def on_darea_motion_notify(self, widget, event):
        if self.adjust_timeout and  self.t_x1 <= event.x <= self.t_x2:
            self.timeout_offset = float(event.x-self.t_x1)/(self.t_x2 - self.t_x1)
            self.queue_draw()
            return
        cursor = self.cursor
        self.cursor = self.xy2move(event.x, event.y, self.who)[:3]
        if self.cursor[0] == 0 and self.hexist[self.cursor[1]][self.cursor[2]] != 0:
            self.cursor = (-1, -1, -1)
        if self.cursor[0] == 1 and self.vexist[self.cursor[1]][self.cursor[2]] != 0:
            self.cursor = (-1, -1, -1)
        if self.cursor != cursor:
            self.queue_draw()
        if self.human + self.robot == 25 and self.who == -1:
            self.who = -2
            if self.human > self.robot:
                s = 'You win.\nGood job!'
            else:
                s = 'I win!\nHahaha'
            msg = gtk.MessageDialog(self, gtk.DIALOG_MODAL, gtk.MESSAGE_INFO, gtk.BUTTONS_CLOSE, s)
            msg.run()
            msg.destroy()

    def on_darea_leave_notify(self, widget, event):
        self.cursor = (-1, -1, -1)
        self.queue_draw()

    def draw_board(self):
        orgx, orgy = self.GridSize, self.GridSize
        w, h = self.darea.get_size_request()
        size = (min(w, h) - (orgx + orgy)) / 5
        cr = self.darea.window.cairo_create()
        cr.rectangle(0, 0, w, h)
        if not self.thinking:
            cr.set_source_rgb(240 / 255.0, 248 / 255.0, 255 / 255.0)
        else:
            cr.set_source_rgb(229 / 255.0, 252 / 255.0, 163 / 255.0)
        cr.fill()
        # draw boxes
        for i in xrange(5):
            for j in xrange(5):
                if self.belong[i][j] == 0:  # human captured
                    cr.rectangle(orgx + size * j, orgy + size * i, size, size)
                    cr.set_source_rgb(250 / 255.0, 199 / 255.0, 199 / 255.0)
                    cr.fill()
                elif self.belong[i][j] == 1:  # robot captured
                    cr.rectangle(orgx + size * j, orgy + size * i, size, size)
                    cr.set_source_rgb(0 / 255.0, 191 / 255.0, 255 / 255.0)
                    cr.fill()
        # draw lines
        cr.set_line_width(2)
        for i in xrange(6):
            for j in xrange(5):
                if self.hexist[i][j] == 1:  # human
                    cr.set_source_rgb(1, 0, 0)
                    cr.move_to(orgx+size*j, orgy+size*i)
                    cr.line_to(orgx+size*(j+1), orgy+size*i)
                    cr.stroke()
                elif self.hexist[i][j] == 2:  # robot
                    cr.set_source_rgb(0, 0, 1)
                    cr.move_to(orgx+size*j, orgy+size*i)
                    cr.line_to(orgx+size*(j+1), orgy+size*i)
                    cr.stroke()
        for i in xrange(5):
            for j in xrange(6):
                if self.vexist[i][j] == 1: # human
                    cr.set_source_rgb(1, 0, 0)
                    cr.move_to(orgx+size*j, orgy+size*i)
                    cr.line_to(orgx+size*j, orgy+size*(i+1))
                    cr.stroke()
                elif self.vexist[i][j] == 2: # robot
                    cr.set_source_rgb(0, 0, 1)
                    cr.move_to(orgx+size*j, orgy+size*i)
                    cr.line_to(orgx+size*j, orgy+size*(i+1))
                    cr.stroke()
        # draw cursor
        cr.set_line_width(2)
        if self.cursor != (-1, -1, -1):
            if self.cursor[0] == 0: # horizon
                cr.set_source_rgb(1, 215/255.0, 0)
                cr.move_to(orgx+size*self.cursor[2], orgy+size*self.cursor[1])
                cr.line_to(orgx+size*(self.cursor[2]+1), orgy+size*self.cursor[1])
                cr.stroke()
            else: # vertical
                cr.set_source_rgb(1, 215/255.0, 0)
                cr.move_to(orgx+size*self.cursor[2], orgy+size*self.cursor[1])
                cr.line_to(orgx+size*self.cursor[2], orgy+size*(self.cursor[1]+1))
                cr.stroke()
        # draw points
        cr.set_source_rgb(0, 0, 0)
        cr.set_line_width(6)
        for i in xrange(6):
            for j in xrange(6):
                cr.arc(orgx+size*i, orgy+size*j, 1, 0, 2*math.pi)
                cr.stroke()
        # draw timeout scroll
        cr.set_line_width(3)
        cr.set_source_rgb(191 / 255.0, 191 / 255.0, 191 / 255.0)
        cr.move_to(self.t_x1, self.t_y)
        cr.line_to(self.t_x2, self.t_y)
        cr.stroke()
        cr.set_line_width(8)
        cr.set_source_rgb(self.timeout_offset, 1-self.timeout_offset, 1)
        cr.move_to(self.t_x1+(self.t_x2-self.t_x1)*self.timeout_offset, self.t_y-10)
        cr.line_to(self.t_x1+(self.t_x2-self.t_x1)*self.timeout_offset, self.t_y+10)
        cr.stroke()

        gc = self.darea.get_style().fg_gc[gtk.STATE_NORMAL]
        gc.set_rgb_fg_color(gtk.gdk.color_parse('#000000'))
        layout = self.darea.create_pango_layout(str(int(10 + 60 * self.timeout_offset)))
        layout.set_font_description(pango.FontDescription('Sans 10'))
        self.darea.window.draw_layout(gc, self.t_x1+(self.t_x2-self.t_x1)/2-10, self.t_y + 20, layout)
        layout = self.darea.create_pango_layout("Fast")
        layout.set_font_description(pango.FontDescription('Sans 10'))
        self.darea.window.draw_layout(gc, self.t_x1-30, self.t_y - 30, layout)
        layout = self.darea.create_pango_layout("Slow")
        layout.set_font_description(pango.FontDescription('Sans 10'))
        self.darea.window.draw_layout(gc, self.t_x2, self.t_y - 30, layout)
        layout = self.darea.create_pango_layout("Easy")
        layout.set_font_description(pango.FontDescription('Sans 10'))
        self.darea.window.draw_layout(gc, self.t_x1-30, self.t_y + 20, layout)
        layout = self.darea.create_pango_layout("Hard")
        layout.set_font_description(pango.FontDescription('Sans 10'))
        self.darea.window.draw_layout(gc, self.t_x2, self.t_y + 20, layout)

    def on_about_activate(self, widget):
        about = gtk.AboutDialog()
        about.set_program_name(self.myname)
        about.set_version(self.version)
        about.set_comments('A simple Dots and Boxes AI.')
        about.set_copyright('Written by:\nYimmon Zhuang\n<yimmon.zhuang@gmail.com>')
        about.run()
        about.destroy()

    def xy2move(self, x, y, who):
        # outside
        x, y = int(x), int(y)
        orgx, orgy = self.GridSize, self.GridSize
        w, h = self.darea.get_size_request()
        size = int((min(w, h) - (orgx+orgy)) / 5)

        if x+orgx < size or y+orgy < size \
            or (x-orgx)/size > 5 or (y-orgy)/size > 5 \
            or (x < size and y < size) or (x < size and y/size > 5) \
            or (x/size > 5 and y < size) or (x/size > 5 and y/size > 5):
                return (-1, -1, -1, who)
        # left margin
        if x < size:
            return (1, y/size-1, 0, who)
        # right (*margin)
        if x > size*(5+1):
            return (1, y/size-1, 5, who)
        # up (*margin)
        if y < size:
            return (0, 0, x/size-1, who)
        # down (*margin)
        if y > size*(5+1):
            return (0, 5, x/size-1, who)
        # inside
        zero = 1e-5
        x1, y1 = float(x/size*size), float(y/size*size)
        x2, y2 = float(x1+size), y1
        x3, y3 = x1, y1+size
        x4, y4 = x2, y3
        p1x, p1y = x4-x1, y4-y1
        p2x, p2y = x-x1, y-y1
        p3x, p3y = x2-x3, y2-y3
        p4x, p4y = x - x3, y - y3
        c1 = (p1x*p2y) - (p1y*p2x)
        c2 = (p3x*p4y) - (p3y*p4x)
        # on diagonal
        if (c1 >= -zero and c1 <= zero) or (c1 >= -zero and c1 <= zero):
            return (-1, -1, -1, who)
        # up part
        if (c1 < 0 and c2 < 0):
            return (0, y/size-1, x/size-1, who)
        # right part
        if (c1 < 0 and c2 > 0):
            return (1, y/size-1, x/size, who)
        # down part
        if (c1 > 0 and c2 > 0):
            return (0, y/size, x/size-1, who)
        # left part
        if (c1 > 0 and c2 < 0):
            return (1, y/size-1, x/size-1, who)
        return (-1, -1, -1, who)

    def change(self, move):
        x, y = move[1], move[2]
        if move[0] == 0: # horizon
            if x > 0:
                if self.degree[x-1][y] == 1:
                    return False
            if x < 5:
                if self.degree[x][y] == 1:
                    return False
        else: # vertical
            if y > 0:
                if self.degree[x][y-1] == 1:
                    return False
            if y < 5:
                if self.degree[x][y] == 1:
                    return False
        return True


    def num2move(self, value, who, step=-1):
        ty, x, y = 1, -1, -1
        if (value&(1<<31)) != 0:
            ty = 0 # horizon
        for i in range(5)[::step]:
            for j in range(6)[::step]:
                if (value&1) == 1:
                    if ty == 0: x, y = j, i
                    else: x, y = i, j
                    break
                value >>= 1
            if x != -1:
                break
        return (ty, x, y, who)

    def move(self, move, change_record = True):
        if len(move) != 4:
            move = self.num2move(move[0], move[1])
        flag, x, y, who = 0, move[1], move[2], move[3]
        if move[0] == 0: # horizon
            if self.hexist[x][y]:
                return
            self.hexist[x][y] = who+1
            if x > 0:
                self.degree[x-1][y] -= 1
                if self.degree[x-1][y] == 0:
                    self.belong[x-1][y] = who
                    if who == 0: self.human += 1
                    else: self.robot += 1
                    flag = 1
            if x < 5:
                self.degree[x][y] -= 1
                if self.degree[x][y] == 0:
                    self.belong[x][y] = who
                    if who == 0: self.human += 1
                    else: self.robot += 1
                    flag = 1
        else: # vertical
            if self.vexist[x][y]:
                return
            self.vexist[x][y] = who+1
            if y > 0:
                self.degree[x][y-1] -= 1
                if self.degree[x][y-1] == 0:
                    self.belong[x][y-1] = who
                    if who == 0: self.human += 1
                    else: self.robot += 1
                    flag = 1
            if y < 5:
                self.degree[x][y] -= 1
                if self.degree[x][y] == 0:
                    self.belong[x][y] = who
                    if who == 0: self.human += 1
                    else: self.robot += 1
                    flag = 1
        if change_record:
            del self.record[self.moves:]
            self.record.append(move)
            self.moves += 1
        if self.human + self.robot == 5*5:
            self.who = -1
            self.queue_draw()
            #def emit():
                #self.darea.emit('motion-notify-event', gtk.gdk.Event(gtk.gdk.MOTION_NOTIFY))
            #glib.timeout_add(100, emit)
            return
        if not flag:
            self.who ^= 1
            self.turn+=1


class Robot(threading.Thread):
    def __init__(self, dab):
        super(Robot, self).__init__()
        self.setDaemon(True)
        self.dab = dab

    def run(self):
        while self.dab.who == 1:
            self.dab.thinking = True
            self.dab.queue_draw()
            if self.dab.first == 0:
                s0, s1 = self.dab.human, self.dab.robot
            else:
                s1, s0 = self.dab.human, self.dab.robot
            now = 0
            if self.dab.who != self.dab.first:
                now = 1
            h, v = 0, 0
            for move in self.dab.record:
                x, y = move[1], move[2]
                if move[0] == 0:
                    v |= (1<<(y*6+x))
                else:
                    h |= (1<<(x*6+y))
            #algorithm = "alphabeta"
            #algorithm = "uct"
            #algorithm = "uctann"
            #algorithm = "quct"
            algorithm = "quctann"
            timeout = int(10 + 60 * self.dab.timeout_offset) * 1000
            s = socket.create_connection(("0.0.0.0", 12345))
            arg = {"id": int(time.time()), "method": "Server.MakeMove",
                   "params": [{"Algorithm": algorithm,
                               "Board": {"H": h, "V": v, "S": [s0, s1], "Now": now, "Turn": self.dab.turn},
                              "Timeout": timeout}]}
            data = simplejson.dumps(arg).encode()
            s.sendall(data)
            data = s.recv(1024).decode()
            s.close()
            res = simplejson.loads(data)
            ms = (res["result"]["H"], res["result"]["V"])
            moves = []
            for i in range(2):
                for n in range(30):
                    if ((1<<n)&ms[i]) != 0:
                        moves.append(self.dab.num2move(((1<<n)|(i<<31)), 1, 1))
            while len(moves) > 1:
                for m in moves:
                    if not self.dab.change(m):
                        self.dab.move(m)
                        moves.remove(m)
                        break
            self.dab.move(moves[0])
            moves.remove(moves[0])
            self.dab.thinking = False
            self.dab.queue_draw()


def RunGUI():
    DAB()
    gtk.main()

if __name__ == '__main__':
    RunGUI()

