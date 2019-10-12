# Settings.py

import sys
import configparser
import curses
from optparse import OptionParser
import os


# Actual
#user_prefs_filename = os.path.expanduser( '~/.umergerc' )
#etc_prefs_filename = '/etc/umerge.conf'

# For testing.  I used the ini extension so emacs uses the right mode.
user_prefs_filename = 'umergerc.ini'
etc_prefs_filename = 'umerge.conf'


class Settings:

    def __init__( self ):
        self.parser = None

        self.etc_prefs = None
        self.user_prefs = None

        self.num_colors = -1

        self.main_prefs = {}


    def __load_etc_prefs( self ):
        if not os.path.exists( etc_prefs_filename ):
            return

        # Need exception handling here?
        self.etc_prefs = configparser.SafeConfigParser()
        self.etc_prefs.read( etc_prefs_filename )


    def __load_user_prefs( self ):
        if not os.path.exists( user_prefs_filename ):
            return

        # Need exception handling here?
        self.user_prefs = configparser.SafeConfigParser()
        self.user_prefs.read( user_prefs_filename )


    def initialize_prefs( self, canvas_max_colors ):
        self.canvas_max_colors = canvas_max_colors

        # FIXME
        #self.__load_etc_prefs()
        #self.__load_user_prefs()

        colors_requested = self.__determine_colors_requested()
        #print( 'colors requested:', colors_requested )

        # none, colors8, or colors256
        color_section = self.__color_section( colors_requested )

        self.__use_hardcoded_defaults( color_section )
        # FIXME
        #self.__use_prefs( self.etc_prefs, color_section )
        #self.__use_prefs( self.user_prefs, color_section )
        self.__use_command_line_prefs( colors_requested )

        #return self.args


# If we have color='none', do we still use curses colors to do the
# reverse video? Do we still need a color section? Would we still want
# to do light on dark or dark on light through curses? Or just
# hardcode the theme I want and leave it at that? Curses does do
# blinking, bold, and stuff like that. See what is possible with just
# a monochrome terminal.



    def __use_prefs( self, config, color_section ):
        for key in config.options( 'general' ):
            #print( 'key:', key )
            value = config.get( 'general', key )
            #print( 'value:', value )
            self.main_prefs[key] = config.get( 'general', key )
        #print( '\n' )
        if color_section != 'none':
            for key in config.options( color_section ):
                #print( 'key:', key )
                value = config.get( color_section, key )
                #print( 'value:', value )
                self.main_prefs[key] = config.get( color_section, key )


    def __get_prefs_colors( self, prefs ):
        if prefs is None or not prefs.has_option( 'general', 'colors' ):
            return None

        colors = prefs.get( 'general', 'colors' )
        #print( 'In __get_prefs_colors:', colors )

        # It will be a string. Convert it to an int.
        # try:
        #     colors_int = int(colors)
        #     return colors_int
        # except:
        #     print( 'Invalid value for colors:', colors )
        #     return None
        return colors


    def __auto_colors( self ):
        if self.canvas_max_colors >= 256:
            return 256
        elif self.canvas_max_colors >= 8:
            return 8
        else:
            return 0 # monochrome


    def __color_section( self, colors ):
        if colors == 8:
            return 'colors8'
        elif colors == 256:
            return 'colors256'
        else:
            return 'none'


    def __determine_colors_requested( self ):
        colors = -1

        #print( '\n' )
        # Hardcoded default
        choice = 'auto'
        #print( '---Hardcoded colors:', choice )

        # etc
        etc_colors = self.__get_prefs_colors( self.etc_prefs )
        if etc_colors is not None:
            choice = str(etc_colors)
        #print( '---etc colors:', str(etc_colors) )

        # .rc
        user_colors = self.__get_prefs_colors( self.user_prefs )
        if user_colors is not None:
            choice = str(user_colors)
        #print( '---user colors:', str(user_colors) )

        # command line
        #print( 'command line colors:', self.options.colors )
        if self.options.colors is not None:
            choice = self.options.colors
        #print( '---CLI colors:', self.options.colors )
        #print( '\n' )

        if choice == 'none':
            colors = 0
        elif choice == '8':
            colors = 8
        elif choice == '256':
            colors = 256
        elif choice == 'auto':
            colors = self.__auto_colors()
        else:
            # fixme: Need error handling here for invalid values
            #print( 'Bogus:xxx%sxxxx' % choice, choice )
            #raise Exception( "Bogus color choice:" + choice )
            pass

        return colors


    def get_value( self, key ):
        #print( 'key:', key )
        try:
            value = self.main_prefs[key]
            #print( 'value:', value )
            while str(value).startswith( '$' ):
                value = self.main_prefs[ value[1:] ]
            return value
        except:
            # Signal an error here somehow.
            return None


    def print_help( self ):
        self.parser.print_help()


    usage = (
'''Usage: %prog [OPTION]... DIRECTORY DIRECTORY
  or:  %prog [OPTION]... CHILD_DIRECTORY ANCESTOR_DIRECTORY CHILD_DIRECTORY''')

    version = (
'''%prog 0.5
Copyright (C) 2010 Michael C. James. All rights reserved.
This software is distributed under the GPL v.2.

This program is provided with NO WARRANTY, to the extent permitted by law.''')


    def parse_command_line( self ):
        self.parser = OptionParser( usage=self.usage, version=self.version )
        parser = self.parser

        # We probably don't need this here
        parser.set_defaults( colors=None )
        parser.add_option( "-c", "--colors", dest="colors",
                           help="number of colors requested" )

        # Unicode printing of tree symbols is the default. If Unicode
        # is not supported, the user will need to force the fallback
        # to ASCII tree symbols.
        parser.set_defaults( ascii=None )
        parser.add_option( "-A", "--ascii", action="store_true", dest="ascii")
        parser.add_option( "-U", "--unicode", action="store_false", dest="ascii")




        parser.add_option( "-a", "--address", dest="host",
                           help="server IP address to which we're connecting" )
        parser.set_defaults( host=None )

        parser.add_option( "-p", "--port", dest="port",
                           help="server port to which we're connecting" )
        parser.set_defaults( port=0 )

        (self.options, args) = parser.parse_args()

        #print( "--colors=", self.options.colors )
        self.cli_colors = self.options.colors

        #print( "ascii=", self.options.ascii )
        self.cli_ascii = self.options.ascii





        #print( 'server_port=', self.options.port )      # int
        if self.options.port != 0:
            self.server_port = int(self.options.port)

        #print( 'server_host=', self.options.host )      # string
        if self.options.host is not None:
            self.server_host = self.options.host

        return args

        # Print a more informative error message here.  Tell the
        # user which option was bad.
        #print( 'Invalid option' )

        #return None


    # Set the correct values for keys for all things that can be set
    # by the comand line.
    def __use_command_line_prefs( self, color_section ):
        #print( 'cli_colors:', self.cli_colors )
        if self.cli_colors is not None:
            self.main_prefs['colors'] = self.cli_colors
        #print( 'final colors:', self.main_prefs['colors'] )

        #print( 'cli_ascii:', self.cli_ascii )
        if self.cli_ascii is not None:
            self.main_prefs['ascii'] = self.cli_ascii
        #print( 'final ascii:', self.main_prefs['ascii'] )


    def __use_hardcoded_defaults( self, color_section ):

        hardcoded_general_prefs = {
            'colors': 'auto',
            'file_merge_program': 'vim', #'emacs'
            "ascii": False
            }

        for key in hardcoded_general_prefs:
            self.main_prefs[key] = hardcoded_general_prefs[key]

        colors256 = {
#             'normal_fg':       curses.COLOR_WHITE,
#             'normal_bg':       curses.COLOR_BLACK,
#             'normal_h_fg':     226,
#             'normal_h_bg':     curses.COLOR_BLACK,
            'normal_dir_arrow_fg':        226,
            'normal_dir_arrow_bg':        curses.COLOR_BLACK,
            'normal_filename_fg':         curses.COLOR_WHITE,
            'normal_filename_bg':         curses.COLOR_BLACK,
            'normal_count_fg':            curses.COLOR_GREEN,
            'normal_count_bg':            curses.COLOR_BLACK,
            'normal_vertical_sep_fg':     curses.COLOR_WHITE,
            'normal_vertical_sep_bg':     curses.COLOR_BLACK,
            'normal_hidden_fg':           237,
            'normal_hidden_bg':           curses.COLOR_BLACK,

            'normal_dir_arrow_h_fg':      226,
            'normal_dir_arrow_h_bg':      240,
            'normal_filename_h_fg':       226,
            'normal_filename_h_bg':       240,
            'normal_count_h_fg':          226,
            'normal_count_h_bg':          240,
            'normal_vertical_sep_h_fg':   226,
            'normal_vertical_sep_h_bg':   240,
            'normal_hidden_h_fg':         240,
            'normal_hidden_h_bg':         237,

#             'differ_fg':       curses.COLOR_BLACK,
#             'differ_bg':       curses.COLOR_CYAN,
#             'differ_h_fg':     226,
#             'differ_h_bg':     54,
            'changed_dir_arrow_fg':       226,
            'changed_dir_arrow_bg':       67,
            'changed_filename_fg':        curses.COLOR_BLACK,
            'changed_filename_bg':        67,
            'changed_count_fg':           curses.COLOR_RED,
            'changed_count_bg':           67,
            'changed_vertical_sep_fg':    curses.COLOR_BLACK,
            'changed_vertical_sep_bg':    67,
            'changed_hidden_fg':          67,
            'changed_hidden_bg':          curses.COLOR_BLACK,

            'changed_dir_arrow_h_fg':     226,
            'changed_dir_arrow_h_bg':     153,
            'changed_filename_h_fg':      226,
            'changed_filename_h_bg':      153,
            'changed_count_h_fg':         226,
            'changed_count_h_bg':         153,
            'changed_vertical_sep_h_fg':  226,
            'changed_vertical_sep_h_bg':  153,
            'changed_hidden_h_fg':        curses.COLOR_BLACK,
            'changed_hidden_h_bg':        153,

#             'only_one_fg':     curses.COLOR_BLACK,
#             'only_one_bg':     curses.COLOR_GREEN,
#             'only_one_h_fg':   226,
#             'only_one_h_bg':   curses.COLOR_GREEN,
            'inserted_dir_arrow_fg':      226,
            'inserted_dir_arrow_bg':      108,
            'inserted_filename_fg':       curses.COLOR_BLACK,
            'inserted_filename_bg':       108,
            'inserted_count_fg':          curses.COLOR_MAGENTA,
            'inserted_count_bg':          108,
            'inserted_vertical_sep_fg':   curses.COLOR_BLACK,
            'inserted_vertical_sep_bg':   108,
            'inserted_hidden_fg':         108,
            'inserted_hidden_bg':         curses.COLOR_BLACK,

            'inserted_dir_arrow_h_fg':    226,
            'inserted_dir_arrow_h_bg':    curses.COLOR_GREEN,
            'inserted_filename_h_fg':     226,
            'inserted_filename_h_bg':     curses.COLOR_GREEN,
            'inserted_count_h_fg':        226,
            'inserted_count_h_bg':        curses.COLOR_GREEN,
            'inserted_vertical_sep_h_fg': 226,
            'inserted_vertical_sep_h_bg': curses.COLOR_GREEN,
            'inserted_hidden_h_fg':       curses.COLOR_BLACK,
            'inserted_hidden_h_bg':       curses.COLOR_GREEN,

            'removed_dir_arrow_fg':       226,
            'removed_dir_arrow_bg':       61,
            'removed_filename_fg':        45,
            'removed_filename_bg':        61,
            'removed_count_fg':           curses.COLOR_WHITE,
            'removed_count_bg':           61,
            'removed_vertical_sep_fg':    curses.COLOR_WHITE,
            'removed_vertical_sep_bg':    61,
            'removed_hidden_fg':          17,
            'removed_hidden_bg':          curses.COLOR_BLACK,

            'removed_dir_arrow_h_fg':     226,
            'removed_dir_arrow_h_bg':     curses.COLOR_MAGENTA,
            'removed_filename_h_fg':      curses.COLOR_RED,
            'removed_filename_h_bg':      curses.COLOR_MAGENTA,
            'removed_count_h_fg':         226,
            'removed_count_h_bg':         curses.COLOR_MAGENTA,
            'removed_vertical_sep_h_fg':  226,
            'removed_vertical_sep_h_bg':  curses.COLOR_MAGENTA,
            'removed_hidden_h_fg':        curses.COLOR_BLACK,
            'removed_hidden_h_bg':        17,

            'absent_fg':                  curses.COLOR_BLACK,
            'absent_bg':                  251,
            'absent_h_fg':                251,
            'absent_h_bg':                253,

            'uncompared_fg':              17,
            'uncompared_bg':              curses.COLOR_BLACK,
            'uncompared_h_fg':            curses.COLOR_BLACK,
            'uncompared_h_bg':            17,

            'error_fg':                   curses.COLOR_RED,
            'error_bg':                   curses.COLOR_BLACK,
            'error_h_fg':                 curses.COLOR_BLACK,
            'error_h_bg':                 curses.COLOR_RED,

            'selected_fg':                curses.COLOR_WHITE,
            'selected_bg':                curses.COLOR_BLUE,
            'selected_h_fg':              curses.COLOR_WHITE,
            'selected_h_bg':              curses.COLOR_CYAN,

            'marker_ok_fg':               curses.COLOR_GREEN,
            'marker_ok_bg':               234,
            'marker_merged_fg':           226,
            'marker_merged_bg':           234,
            'marker_resolved_fg':         226,
            'marker_resolved_bg':         234,
            'marker_conflict_fg':         curses.COLOR_RED,
            'marker_conflict_bg':         234,

            'top_line_fg':                curses.COLOR_WHITE,
            'top_line_bg':                236,
            'top_vertical_sep_fg':        240,
            'top_vertical_sep_bg':        236,

            'status_1_fg':                curses.COLOR_WHITE,
            'status_1_bg':                240,
            'status_2_fg':                curses.COLOR_GREEN,
            'status_2_bg':                240,
            'status_3_fg':                curses.COLOR_WHITE,
            'status_3_bg':                240,
            'status_4_fg':                curses.COLOR_BLACK,
            'status_4_bg':                240,

            'prompt_1_fg':                curses.COLOR_WHITE,
            'prompt_1_bg':                curses.COLOR_BLACK,
            'prompt_2_fg':                curses.COLOR_WHITE,
            'prompt_2_bg':                curses.COLOR_BLACK,
            'prompt_3_fg':                curses.COLOR_WHITE,
            'prompt_3_bg':                curses.COLOR_BLACK,
            'prompt_4_fg':                curses.COLOR_WHITE,
            'prompt_4_bg':                curses.COLOR_BLACK,
            }

#         colors256 = {
#             'normal_fg':       curses.COLOR_WHITE,
#             'normal_bg':       curses.COLOR_BLACK,
#             'normal_h_fg':     curses.COLOR_YELLOW,
#             'normal_h_bg':     curses.COLOR_BLACK,

#             'differ_fg':       curses.COLOR_BLACK,
#             'differ_bg':       curses.COLOR_CYAN,
#             'differ_h_fg':     curses.COLOR_YELLOW,
#             'differ_h_bg':     54,

#             'only_one_fg':     curses.COLOR_BLACK,
#             'only_one_bg':     curses.COLOR_GREEN,
#             'only_one_h_fg':   curses.COLOR_YELLOW,
#             'only_one_h_bg':   curses.COLOR_GREEN,

#             'uncompared_fg':   '$normal_h_fg', #17, #240,
#             'uncompared_bg':   curses.COLOR_BLACK,
#             'uncompared_h_fg': curses.COLOR_YELLOW,
#             'uncompared_h_bg': '$normal_h_fg', #17, #240,

#             'error_fg':        curses.COLOR_RED,
#             'error_bg':        curses.COLOR_BLACK,
#             'error_h_fg':      curses.COLOR_YELLOW,
#             'error_h_bg':      curses.COLOR_RED,

#             'selected_fg':     curses.COLOR_MAGENTA,
#             'selected_h_fg':   curses.COLOR_GREEN
#             }

        colors8 = {
            'normal_fg':       curses.COLOR_WHITE,
            'normal_bg':       curses.COLOR_BLACK,
            'normal_h_fg':     curses.COLOR_YELLOW,
            'normal_h_bg':     curses.COLOR_BLACK,

            'differ_fg':       curses.COLOR_BLACK,
            'differ_bg':       curses.COLOR_CYAN,
            #'differ_bg':       cyan,
            'differ_h_fg':     curses.COLOR_YELLOW,
            'differ_h_bg':     5, #curses.COLOR_CYAN,
            #'differ_h_bg':     'bright_cyan',

            'only_one_fg':     curses.COLOR_BLACK,
            'only_one_bg':     curses.COLOR_GREEN,
            'only_one_h_fg':   curses.COLOR_YELLOW,
            'only_one_h_bg':   curses.COLOR_GREEN,

            'uncompared_fg':   curses.COLOR_WHITE,
            'uncompared_bg':   curses.COLOR_BLACK,
            'uncompared_h_fg': curses.COLOR_YELLOW,
            'uncompared_h_bg': curses.COLOR_WHITE,

            'error_fg':        curses.COLOR_RED,
            'error_bg':        curses.COLOR_BLACK,
            'error_h_fg':      curses.COLOR_YELLOW,
            'error_h_bg':      curses.COLOR_RED
            }

        if color_section == 'colors256':
            for key in colors256:
                self.main_prefs[key] = colors256[key]
        elif color_section == 'colors8':
            for key in colors8:
                self.main_prefs[key] = colors8[key]
        else:
            for key in colors256:
                self.main_prefs[key] = None


if __name__ == '__main__':
    settings = Settings()
    args = settings.initialize_prefs( 256 )
