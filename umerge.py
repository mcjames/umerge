#!/usr/bin/env python3

# umerge.py

import Controller
import FileMergeEmacs
import FileMergeVim
import FileOpsPOSIX
import ScreenCurses
import Model2
import Model3
import Settings
import View2
import View3
import os
import sys
import time

import locale
locale.setlocale(locale.LC_ALL, '')


# For debugging
# sys.stdout = open('/dev/pts/2', 'w')
# sys.stderr = sys.stdout


def main():
    try:
        global args
        global canvas
        global settings

        controller = None
        filemerge = None
        view = None
        model = None

        settings.initialize_prefs(canvas.max_colors())

        # Only POSIX systems are supported at this point
        fileops = FileOpsPOSIX.FileOpsPOSIX()

        if len(args) == 2:
            left = args[0]
            right = args[1]

            left = os.path.realpath(left)
            right = os.path.realpath(right)

            # Instantiate the classes and link them together.
            model = Model2.Model(fileops, left, right)
            view = View2.View(canvas, model, settings)
        else:
            # len(args) == 3
            left = args[0]
            middle = args[1]
            right = args[2]

            left = os.path.realpath(left)
            middle = os.path.realpath(middle)
            right = os.path.realpath(right)

            # Instantiate the classes and link them together.
            model = Model3.Model(fileops, left, middle, right)
            view = View3.View(canvas, model, settings)

        requested_filemerge = settings.get_value('file_merge_program')
        # print('file_merge_program:', requested_filemerge)
        if requested_filemerge == 'vim':
            filemerge = FileMergeVim.FileMergeVim()
        else:
            filemerge = FileMergeEmacs.FileMergeEmacs()
        controller = Controller.Controller(model, view, canvas, filemerge)

        # Start the model. It will enumerate and then start comparing.
        model.start_enumerate()

        # Main event loop
        while not controller.need_to_quit:
            if canvas.need_to_resize:
                view.resize()
            model.render_again = False
            view.render()
            if ((model.state() == Model2.STATE_NORMAL
                or model.state() == Model3.STATE_NORMAL)
                    and not model.render_again
                    and not canvas.need_to_resize):
                input = canvas.get_input(0)  # wait indefinitely
            else:
                input = canvas.get_input(1)  # timeout in 1/10 second
            controller.process_input(input)

    finally:
        # print("quitting")
        if controller is not None:
            controller.destroy()
        if filemerge is not None:
            filemerge.destroy()
        if view is not None:
            view.destroy()
        if model is not None:
            model.destroy()


settings = Settings.Settings()
args = settings.parse_command_line()

if len(args) == 2:
    left = args[0]
    right = args[1]

    left = os.path.realpath(left)
    right = os.path.realpath(right)

    # # Need to handle the case of one or both of the directories not
    # # existing, or not being able to be read, or not being directories.
    fail = False
    if not os.path.isdir(left):
        print("%s: %s: Is not a directory" % (sys.argv[0], left))
        fail = True

    if not os.path.isdir(right):
        print("%s: %s: Is not a directory" % (sys.argv[0], right))
        fail = True

    if fail:
        print("See '%s --help'." % sys.argv[0])
        exit(1)

elif len(args) == 3:
    left = args[0]
    middle = args[1]
    right = args[2]

    left = os.path.realpath(left)
    middle = os.path.realpath(middle)
    right = os.path.realpath(right)

    # print("Left directory : ", left)
    # print("Middle directory: ", middle)
    # print("Right directory: ", right)
    # print("-----------------------------------------------")

    # # Need to handle the case of one or both of the directories not
    # # existing, or not being able to be read, or not being directories.
    fail = False
    if not os.path.isdir(left):
        print("%s: %s: Is not a directory" % (sys.argv[0], left))
        fail = True

    if not os.path.isdir(middle):
        print("%s: %s: Is not a directory" % (sys.argv[0], middle))
        fail = True

    if not os.path.isdir(right):
        print("%s: %s: Is not a directory" % (sys.argv[0], right))
        fail = True

    if fail:
        print("See '%s --help'." % sys.argv[0])
        exit(1)

else:
    settings.print_help()
    exit(1)

canvas = ScreenCurses.Canvas()
canvas.wrapper(main)
canvas.destroy()

# I should modify model.destroy() to kill all child processes, but
# haven't done that yet. Do an exit(0) here to clean up everything.
exit(0)
