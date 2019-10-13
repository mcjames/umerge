# View2.py

import Match2
import Model2
import os


# This class takes the data from the model and renders it to the
# canvas. It knows nothing about the libraries that get the actual
# drawing done. It keeps track of where in the data we should be
# grabbing data to draw and keeps track of things like the cursor. It
# has no control capability.


NORMAL_DIR_ARROW = 1
NORMAL_FILENAME = 2
NORMAL_COUNT = 3
NORMAL_VERTICAL_SEP = 4
NORMAL_HIDDEN = 5

NORMAL_DIR_ARROW_H = 6
NORMAL_FILENAME_H = 7
NORMAL_COUNT_H = 8
NORMAL_VERTICAL_SEP_H = 9
NORMAL_HIDDEN_H = 10

CHANGED_DIR_ARROW = 11
CHANGED_FILENAME = 12
CHANGED_COUNT = 13
CHANGED_VERTICAL_SEP = 14
CHANGED_HIDDEN = 15

CHANGED_DIR_ARROW_H = 16
CHANGED_FILENAME_H = 17
CHANGED_COUNT_H = 18
CHANGED_VERTICAL_SEP_H = 19
CHANGED_HIDDEN_H = 20

INSERTED_DIR_ARROW = 21
INSERTED_FILENAME = 22
INSERTED_COUNT = 23
INSERTED_VERTICAL_SEP = 24
INSERTED_HIDDEN = 25

INSERTED_DIR_ARROW_H = 26
INSERTED_FILENAME_H = 27
INSERTED_COUNT_H = 28
INSERTED_VERTICAL_SEP_H = 29
INSERTED_HIDDEN_H = 30

REMOVED_DIR_ARROW = 31
REMOVED_FILENAME = 32
REMOVED_COUNT = 33
REMOVED_VERTICAL_SEP = 34
REMOVED_HIDDEN = 35

REMOVED_DIR_ARROW_H = 36
REMOVED_FILENAME_H = 37
REMOVED_COUNT_H = 38
REMOVED_VERTICAL_SEP_H = 39
REMOVED_HIDDEN_H = 40

ABSENT = 41
ABSENT_H = 42
UNCOMPARED = 43
UNCOMPARED_H = 44
ERROR = 45
ERROR_H = 46
SELECTED = 47
SELECTED_H = 48

# Maybe make MARKER_MERGED and MARKER_RESOLVED the same?
MARKER_OK = 49
MARKER_MERGED = 50
MARKER_RESOLVED = 51
MARKER_CONFLICT = 52

TOP_LINE = 53
TOP_VERTICAL_SEP = 54

# Maybe make STATUS_1 and TOP the same?
STATUS_1 = 55
STATUS_2 = 56
STATUS_3 = 57
STATUS_4 = 58

PROMPT_1 = 59
PROMPT_2 = 60
PROMPT_3 = 61
PROMPT_4 = 62


highlight = {
    NORMAL_FILENAME:    NORMAL_FILENAME_H,
    NORMAL_HIDDEN:      NORMAL_HIDDEN_H,

    CHANGED_FILENAME:   CHANGED_FILENAME_H,
    CHANGED_HIDDEN:     CHANGED_HIDDEN_H,

    INSERTED_FILENAME:  INSERTED_FILENAME_H,
    INSERTED_HIDDEN:    INSERTED_HIDDEN_H,

    REMOVED_FILENAME:   REMOVED_FILENAME_H,
    REMOVED_HIDDEN:     REMOVED_HIDDEN_H,

    ABSENT:             ABSENT_H,
    UNCOMPARED:         UNCOMPARED_H,
    ERROR:              ERROR_H,
    SELECTED:           SELECTED_H,
}

normal_color_list = (NORMAL_DIR_ARROW, NORMAL_FILENAME, NORMAL_COUNT,
                     NORMAL_VERTICAL_SEP, NORMAL_HIDDEN)
changed_color_list = (CHANGED_DIR_ARROW, CHANGED_FILENAME, CHANGED_COUNT,
                      CHANGED_VERTICAL_SEP, CHANGED_HIDDEN)
inserted_color_list = (INSERTED_DIR_ARROW, INSERTED_FILENAME, INSERTED_COUNT,
                       INSERTED_VERTICAL_SEP, INSERTED_HIDDEN)
removed_color_list = (REMOVED_DIR_ARROW, REMOVED_FILENAME, REMOVED_COUNT,
                      REMOVED_VERTICAL_SEP, REMOVED_HIDDEN)

normal_h_color_list = (NORMAL_DIR_ARROW_H, NORMAL_FILENAME_H, NORMAL_COUNT_H,
                       NORMAL_VERTICAL_SEP_H, NORMAL_HIDDEN_H)
changed_h_color_list = (CHANGED_DIR_ARROW_H, CHANGED_FILENAME_H,
                        CHANGED_COUNT_H, CHANGED_VERTICAL_SEP_H,
                        CHANGED_HIDDEN_H)
inserted_h_color_list = (INSERTED_DIR_ARROW_H, INSERTED_FILENAME_H,
                         INSERTED_COUNT_H, INSERTED_VERTICAL_SEP_H,
                         INSERTED_HIDDEN_H)
removed_h_color_list = (REMOVED_DIR_ARROW_H, REMOVED_FILENAME_H,
                        REMOVED_COUNT_H, REMOVED_VERTICAL_SEP_H,
                        REMOVED_HIDDEN_H)

normal_hidden_color_list = [NORMAL_HIDDEN] * 5
normal_hidden_h_color_list = [NORMAL_HIDDEN_H] * 5

changed_hidden_color_list = [CHANGED_HIDDEN] * 5
changed_hidden_h_color_list = [CHANGED_HIDDEN_H] * 5

inserted_hidden_color_list = [INSERTED_HIDDEN] * 5
inserted_hidden_h_color_list = [INSERTED_HIDDEN_H] * 5

removed_hidden_color_list = [REMOVED_HIDDEN] * 5
removed_hidden_h_color_list = [REMOVED_HIDDEN_H] * 5

uncompared_color_list = [UNCOMPARED] * 5
uncompared_h_color_list = [UNCOMPARED_H] * 5

error_color_list = [ERROR] * 5
error_h_color_list = [ERROR_H] * 5


#####

LEFT = 0
MIDDLE = 1
RIGHT = 2


TREE_SYMBOL_CLOSED = 0
TREE_SYMBOL_OPEN = 1
ascii_tree_symbols = ("+", "-")
unicode_tree_symbols = ("\u25B6", "\u25BC")  # right arrow, down arrow


#######################################

class View:

    def __init__(self, canvas, model, settings):

        self.settings = settings

        self.canvas = canvas
        canvas.set_view(self)

        self.__recalculate_sizes()

        self.model = model
        self.cursor = 1
        self.last_displayed_item_row = self.cursor
        self.current = []  # current is at [0], top is at length-1
        self.top = None
        self.spinner = 1
        self.prompt = ''
        self.render_hidden = False

        if self.settings.get_value('ascii'):
            self.tree_symbols = ascii_tree_symbols
        else:
            self.tree_symbols = unicode_tree_symbols

        self.__init_color_pairs()

    def __init_a_color_pair(self, index, fg_name, bg_name):
        fg_number = int(self.settings.get_value(fg_name))
        bg_number = int(self.settings.get_value(bg_name))
        self.canvas.init_color_pair(index, fg_number, bg_number)

    def destroy(self):
        pass

    def current_item(self):
        assert(len(self.current) > 0)
        return self.current[0]

    def reset_cursor_after_delete(self):
        # print('reset_cursor_after_delete()')
        if self.cursor == 1:
            self.__reset_top()
            self.canvas.set_full_refresh()
        else:
            if self.__next_display_line(self.current) is None:
                self.cursor -= 1

    def __reset_top(self):
        # print('reset_top()')
        current = self.top

        # fixme: can len(current) ever be zero?
        # while current is not None and current[0].IsHidden():
        while (current is not None
               and not self.__should_render_item(current[0])):
            current = self.__next_display_line_aux(current)
            # print('path=', current)

        if current is None:
            # scroll up one page
            self.scroll(-self.rows)  # fixme: should be # of items

            # fixme: should be last valid item
            self.cursor = self.last_item_row
        else:
            # Effectively make the next undeleted item the top one
            self.top = current

    def toggle_collapse(self):
        self.current_item().toggle_collapse()

    def __can_scroll_down(self):
        # fixme: not entirely correct
        return self.last_displayed_item_row == self.last_item_row

    # fixme: why does this always return True?
    def __can_scroll_up(self):
        return True

    def scroll(self, number_of_lines):
        # print("\nself.last_displayed_item_row:",
        #       self.last_displayed_item_row)
        # print("self.rows:", self.rows)
        # print("last_item_on_screen:", self.last_displayed_item_row)

        if number_of_lines > 0:
            if not self.__can_scroll_down():
                return

            self.canvas.set_full_refresh()
            i = 0
            while i < number_of_lines:
                # This works well to prevent crashes, but if you
                # page down and the last line is half-way up the
                # screen, it will make the last line "top." It might
                # be nice to execute the loop before setting top, and
                # not change top if we can't scroll a whole page. I'll
                # live with this awhile and see which I like better.
                next = self.__next_display_line(self.top)
                if next is not None:
                    self.top = next
                i += 1
            # print('self.cursor:', self.cursor)
            # print('self.last_displayed_item_row:',
            #       self.last_displayed_item_row)
            # fixme: this doesn't work here. This used to work, and I
            # broke it.
            if self.cursor > self.last_displayed_item_row:
                self.cursor = self.last_displayed_item_row
        else:
            if not self.__can_scroll_up():
                return

            self.canvas.set_full_refresh()
            number_of_lines = -number_of_lines
            i = 0
            while i < number_of_lines:
                prev = self.__prev_display_line(self.top)
                if prev is not None:
                    self.top = prev
                i += 1

    def cursor_up(self):
        if self.__prev_display_line(self.current) is not None:
            if self.cursor > 1:
                self.cursor -= 1
            else:
                self.scroll(-1)
        else:
            # fixme: Flash the console here.
            # print('\a')
            pass

    def cursor_down(self):
        if self.__next_display_line(self.current) is not None:
            if self.cursor < self.last_displayed_item_row:
                self.cursor += 1
            else:
                self.scroll(1)
        else:
            # fixme: Flash the console here.
            # print('\a')
            pass

    def render(self):

        self.__render_items()
        self.__render_roots_line()
        self.__render_status_line()
        self.__render_command_line()

        self.canvas.refresh()

    def __render_roots_line(self):
        # Render the top line here with the model.top directories
        # on it. In order to get this right, I'll have to clean up
        # things to compute the rest of the lines right if I start on
        # row 1 rather than row 0.
        left = self.model.top.left
        if len(left) > self.left_item_width:
            left = left[:self.left_item_width]
        elif len(left) < self.left_item_width:
            left += ' ' * (self.left_item_width - len(left))

        right = self.model.top.right
        if len(right) > self.right_item_width:
            right = right[:self.right_item_width]
        elif len(right) < self.right_item_width:
            right += ' ' * (self.right_item_width - len(right))

        col = 0
        self.canvas.draw_text(0, col, left, TOP_LINE)
        col += len(left)

        self.canvas.draw_text(0, col, '|', TOP_VERTICAL_SEP)
        col += 1

        self.canvas.draw_text(0, col, right, TOP_LINE)
        col += len(right)

    def __render_status_line(self):
        test_string = " " * self.cols

        # print('view.cols=', self.cols)
        amount_over = len(test_string) - self.cols
        # print('amount_over=', amount_over)
        if amount_over > 0:
            test_string = test_string[:-amount_over]

        # print('final length=', len(test_string))
        self.canvas.draw_text(self.status_row, 0, test_string, STATUS_1)

        spinner = ' |/-\\'
        if self.model.state() == Model2.STATE_NORMAL:
            self.spinner = 0
        else:
            self.spinner += 1
            if self.spinner == 5:  # len(spinner)
                self.spinner = 1
            self.canvas.draw_text(self.status_row, 50, spinner[self.spinner],
                                  STATUS_2)

    def __render_command_line(self):
        # Curses cannot display in the last character of the last
        # line, so shorten it by amount_over + 1.

        test_string = ' ' * self.cols

        # print('view.cols=', self.cols)
        amount_over = len(test_string) - (self.cols - 1)
        # print('amount_over=', amount_over)
        if amount_over > 0:
            test_string = test_string[:-amount_over]

        # print('final length=', len(test_string))

        self.canvas.draw_text(self.command_row, 0, test_string, PROMPT_1)
        if self.model.state() == Model2.STATE_ENUMERATING:
            self.canvas.draw_text(self.command_row, 0,
                                  'Enumerating directories and files...',
                                  PROMPT_1)
        else:
            self.canvas.draw_text(self.command_row, 0, self.prompt,
                                  PROMPT_1)

    def __render_items(self):
        # print('\nIn View.render()')
        if self.model.state() == Model2.STATE_ENUMERATING:
            self.canvas.clear()
        else:
            with self.model.lock():
                # print('start of render')
                self.canvas.clear()

                if self.top is None:
                    self.top = [self.model.top.children[0], self.model.top]

                row = 1
                current = self.top

                # print('rows=', self.rows)
                # print('cols=', self.cols)
                while row <= self.last_item_row:
                    if row == self.cursor:
                        self.current = current
                    self.__render_path_item(row, current)
                    self.last_displayed_item_row = row
                    row += 1
                    current = self.__next_display_line(current)
                    # print('current=', current.left)
                    if current is None:
                        break

                while row <= self.last_item_row:
                    self.__render_empty_item(row)
                    row += 1
                # print('end of render')

    def resize(self):
        # print('\nResizing...')
        self.canvas.resize()
        self.__recalculate_sizes()
        self.canvas.set_full_refresh()

    def __recalculate_sizes(self):
        # Now recompute the locations of various things.
        self.rows = self.canvas.rows
        self.cols = self.canvas.cols

        # print('  new rows=', self.rows)
        # print('  new cols=', self.cols)

        # fixme: If there are less than four rows, don't render and just
        # print an error that the display is too small.
        self.roots_row = 0
        self.first_item_row = 0  # fixme: make this = 1
        self.last_item_row = self.rows - 3
        self.status_row = self.rows - 2
        self.command_row = self.rows - 1

        # We use self.cols - 1 since there is a vertical bar in the middle.
        self.left_item_width = (self.cols - 1) // 2
        self.right_item_width = self.left_item_width
        if (self.cols - 1) % 2:
            self.left_item_width += 1
        assert(self.left_item_width + self.right_item_width + 1 == self.cols)
        # print('left_item_width =', self.left_item_width)
        # print('right_item_width=', self.right_item_width)

    def __render_merge_item_status(self, row, col, item):
        status = item.resolution_status
        if status == ' ' or status == 'a' or status == 'b':
            color = MARKER_OK
        elif status == 'm':
            color = MARKER_MERGED
        elif status == 'r':
            color = MARKER_RESOLVED
        elif status == 'c':
            color = MARKER_CONFLICT
        else:
            assert(False)

        self.__render_line(row, col,
                           item.resolution_status, color)

        return col + 1

    def __render_path_item(self, row, path_item):
        indention = 4 * (len(path_item) - 2)
        item = path_item[0]

        colors = self.__fixed_compute_colors(row == self.cursor, item)
        # print("len(colors)=", len(colors))
        col = 0

        # if True: #self.model.merge_done:
        #     col += self.__render_merge_item_status(row, col, item)

        col += self.__render_partial_path_item(row, col, item, LEFT,
                                               indention, colors[0],
                                               colors[3])

        self.__render_line(row, col, '|', colors[1])
        col += 1

        # col += self.__render_partial_path_item(row, col, item, MIDDLE,
        #                                         indention, colors[2],
        #                                         colors[5])

        # self.__render_line(row, col, '|', colors[3])
        # col += 1

        col += self.__render_partial_path_item(row, col, item, RIGHT,
                                               indention, colors[2],
                                               colors[3])

#         col = 0
#         self.__render_line(row, col, item.resolution_status,
#               self.__resolution_status_to_color_pair(item.resolution_status))
#         col += 1

#         self.__render_line(row, col, left_name[:-1], colors[0])
#         col += len(left_name) - 1

#         self.__render_line(row, col, '|', lm_bar_color)
#         col += 1

#         self.__render_line(row, col, middle_name, colors[1])
#         col += len(middle_name)

#         self.__render_line(row, col, '|', mr_bar_color)
#         col += 1

#         self.__render_line(row, col, right_name, colors[2])

    ##############################
    #
#     def __create_partial_path_item(self,
#                                     path,
#                                     indention,
#                                     collapse,
#                                     num_diffs,
#                                     desired_size):
#         if path == None:
#             result = " " * desired_size
#         else:
#             result = " " * indention

#             if os.path.isdir(path):
#                 if collapse:
#                     result += '\u25B6 ' #'- '
#                 else:
#                     result += '\u25BC ' #'o '
#             else:
#                 result += '  '

#             result += os.path.basename(path)
#             #if not os.path.isdir(path):
#             if num_diffs is not None:
#                 result += "---" + str(num_diffs)
#             if len(result) < desired_size:
#                 result += " " * (desired_size - len(result))
#             elif len(result) > desired_size:
#                 result = result[:-(len(result) - desired_size)]

#         return result

    ##############################
    #
    def __render_partial_path_item(self,
                                   row,
                                   col,
                                   item,
                                   lmr,
                                   indention,
                                   color,
                                   selection_color):
        # orig_col = col

        if lmr == LEFT:
            name = item.left
            num_diffs = item.num_diffs
            desired_size = self.left_item_width
            if False:  # self.model.merge_done:
                desired_size -= 1
        else:
            name = item.right
            num_diffs = item.num_diffs
            desired_size = self.right_item_width

        # fixme hack

        # name = item.left
        # num_diffs = item.num_diffs
        # desired_size = self.left_item_width
        # if True: #self.model.merge_done:
        #     desired_size -= 1

        # orig_desired_size = desired_size

        if name is None:
            self.__render_line(row, col, ' ' * desired_size, color)
            return desired_size
        else:
            # indention, arrow, name, postname_space, count, postcount_space]
            format_list = [' '] * 6

            # first_part = ''
            # second_part = ''

            # indention_part = ' ' * indention
            format_list[0] = ' ' * indention

            if os.path.isdir(name):
                if item.collapse:
                    collapse_part = self.tree_symbols[TREE_SYMBOL_CLOSED] + " "
                else:
                    collapse_part = self.tree_symbols[TREE_SYMBOL_OPEN] + " "
            else:
                collapse_part = "  "
            format_list[1] = collapse_part

            # name_part = os.path.basename(name)
            format_list[2] = os.path.basename(name)

            if lmr == LEFT and num_diffs is not None:
                # count_part = '---' + str(num_diffs)
                if num_diffs == 0:
                    count_part = '='
                else:
                    count_part = str(num_diffs)
            else:
                count_part = ''
            format_list[4] = count_part

            format_list = self.__reformat_partial_item(desired_size,
                                                       format_list)
            return self.__render_format_list(row, col, format_list, color,
                                             item.selected, selection_color)

    ##############################
    #
    def __render_format_list(self, row, col, format_list, color, selected,
                             selection_color):
        if color == NORMAL_FILENAME:  # fixme: make categories
            colors = normal_color_list
        elif color == CHANGED_FILENAME:
            colors = changed_color_list
        elif color == INSERTED_FILENAME:
            colors = inserted_color_list
        elif color == REMOVED_FILENAME:
            colors = removed_color_list
        elif color == UNCOMPARED:
            colors = uncompared_color_list
        elif color == ERROR:
            colors = error_color_list
        elif color == NORMAL_FILENAME_H:
            colors = normal_h_color_list
        elif color == CHANGED_FILENAME_H:
            colors = changed_h_color_list
        elif color == INSERTED_FILENAME_H:
            colors = inserted_h_color_list
        elif color == REMOVED_FILENAME_H:
            colors = removed_h_color_list
        elif color == UNCOMPARED_H:
            colors = uncompared_h_color_list
        elif color == ERROR_H:
            colors = error_h_color_list

        elif color == NORMAL_HIDDEN:
            colors = normal_hidden_color_list
        elif color == NORMAL_HIDDEN_H:
            colors = normal_hidden_h_color_list

        elif color == CHANGED_HIDDEN:
            colors = changed_hidden_color_list
        elif color == CHANGED_HIDDEN_H:
            colors = changed_hidden_h_color_list

        elif color == INSERTED_HIDDEN:
            colors = inserted_hidden_color_list
        elif color == INSERTED_HIDDEN_H:
            colors = inserted_hidden_h_color_list

        elif color == REMOVED_HIDDEN:
            colors = removed_hidden_color_list
        elif color == REMOVED_HIDDEN_H:
            colors = removed_hidden_h_color_list

        else:
            # print('bogus color=', color)
            # print('format_list=', format_list)
            assert False
            return

        total = 0
        self.__render_line(row, col + total, format_list[0], colors[1])
        total += len(format_list[0])

        self.__render_line(row, col + total, format_list[1], colors[0])
        total += len(format_list[1])

        if selected:
            self.__render_line(row, col + total, format_list[2],
                               selection_color)
        else:
            self.__render_line(row, col + total, format_list[2], colors[1])
        total += len(format_list[2])

        self.__render_line(row, col + total, format_list[3], colors[1])
        total += len(format_list[3])

        self.__render_line(row, col + total, format_list[4], colors[2])
        total += len(format_list[4])

        self.__render_line(row, col + total, format_list[5], colors[1])
        total += len(format_list[5])

        return total

    ##############################
    #
    def __reformat_partial_item(self, desired_size, format_list):
        # print('start=', format_list)

        # first_part = indention_part + collapse_part + name_part
        first_part = format_list[0] + format_list[1] + format_list[2]
        # second_part = count_part
        second_part = ' ' + format_list[4] + ' '  # format_list[5]

        amount_over = len(first_part) + len(second_part) - desired_size
        if amount_over > 0:
            desired_first_len = desired_size - len(second_part)

            len_0 = len(format_list[0])
            len_01 = len(format_list[0] + format_list[1])
            len_012 = len(first_part)

            if len_0 >= desired_first_len:
                format_list[0] = format_list[0][:desired_first_len]
                format_list[1] = ''
                format_list[2] = ''
            elif len_01 >= desired_first_len:
                # format_list[0] = format_list[0]

                new_1_len = desired_first_len - len_0
                format_list[1] = format_list[1][:new_1_len]

                format_list[2] = ''
            elif len_012 > desired_first_len:
                # format_list[0] = format_list[0]
                # format_list[1] = format_list[1]

                new_2_len = desired_first_len - len_01
                format_list[2] = format_list[2][:new_2_len]

            # first_part = first_part[:-amount_over]
        elif amount_over < 0:
            add = -amount_over
            format_list[3] = ' ' * (add + 1)

        # print('final=', format_list)

        # new_len = (len(format_list[0]) + len(format_list[1])
        #           + len(format_list[2]) + len(format_list[3])
        #           + len(format_list[4]) + len(format_list[5]))
        # print('desired_size=', desired_size)
        # print('new_len=', new_len)

        return format_list

#         amount_over = len(first_part) + len(second_part) - desired_size
#         if amount_over > 0:
# #             if item.selected:
# #                 self.__render_line(row, col, result, SELECTED)
# #             else:
# #                 self.__render_line(row, col, result, color)
#             self.__render_line(row, col, first_part[:-amount_over],
#                                 color)
#             self.__render_line(row, col + len(first_part) - amount_over,
#                                 second_part, color)
#         else:
#             self.__render_line(row, col, first_part, color)
#             col += len(first_part)
#             self.__render_line(row, col, second_part, color)
#             col += len(second_part)
#             needed = desired_size - (len(first_part) + len(second_part))
#             self.__render_line(row, col,' ' * needed,color)

#             self.__render_line(row, col, ' ' * indention, color)
#             col += indention
#             first_part = ' ' * indention

#             if os.path.isdir(name):
#                 if item.collapse:
#                     self.__render_line(row, col, '\u25B6 ', color)
#                     #result += '\u25B6 ' #'- '
#                 else:
#                     self.__render_line(row, col, '\u25BC ', color)
#                     #result += '\u25BC ' #'o '
#             else:
#                 self.__render_line(row, col, ' ', color)
#                 #result += '  '
#             col += 1
#             first_part += ' '

#             result = ''
#             result += os.path.basename(name)
#             if item.selected:
#                 self.__render_line(row, col, result, SELECTED)
#             else:
#                 self.__render_line(row, col, result, color)
#             print('result 1=', result)
#             col += len(result)
#             first_part += result

#             second_part = ''
#             if num_diffs is not None:
#                 added = "---" + str(num_diffs)
#                 second_part += added
#                 col += len(added)
#                 second_part = added
#             desired_size -= len(tally)
#             print('desired_size=', desired_size)
#             if len(result) < desired_size:
#                 result += " " * (desired_size - len(result))
#             elif len(result) > desired_size:
#                 result = result[:-(len(result) - desired_size)]

#            print('result 2=', result)
#            self.__render_line(row, col, result, color)

#        return orig_desired_size

    def __should_render_item(self, item):
        if self.render_hidden:
            return not item.IsHidden()
        else:
            return not item.IsHidden() and not item.hidden

    def __fixed_compute_colors(self, is_cursor, item):
        colors = self.__compute_colors(item)
        # print('colors=', colors)
        new_colors = [None] * 4

        new_colors[0] = colors[0]
        new_colors[1] = colors[1]
        new_colors[2] = colors[1]
        new_colors[3] = SELECTED

        if is_cursor:
            for i in range(len(new_colors)):
                new_colors[i] = highlight[new_colors[i]]
            # new_colors[5] = SELECTED_H

        return new_colors

    def __compute_colors(self, item):
        # return [INSERTED_FILENAME, ABSENT, INSERTED_FILENAME]
        # This isn't the right way to deal with uncompared things, but I
        # do need to handle them.
        if item.FilesAreUncompared():
            # fixme: I could display green too
            return [UNCOMPARED, UNCOMPARED]

        if item.hidden:
            # fixme: I could display green too
            return [NORMAL_HIDDEN, NORMAL_HIDDEN]

        if item.HasError():
            return [ERROR, ERROR]

        # count = item.count()

        if item.HasError():
            return [ERROR, ERROR]
        elif item.left is None and item.right is not None:
            return [NORMAL_FILENAME, INSERTED_FILENAME]
            # return [ABSENT, CHANGED_FILENAME]
        elif item.left is not None and item.right is None:
            return [INSERTED_FILENAME, NORMAL_FILENAME]
            # return [CHANGED_FILENAME, ABSENT]
        elif item.FilesAreDifferent():
            return [CHANGED_FILENAME, CHANGED_FILENAME]
        elif item.FilesAreUncompared():
            return [UNCOMPARED, UNCOMPARED]
        else:
            # print finalline
            return [NORMAL_FILENAME, NORMAL_FILENAME]

        # if count == 3:
        #     result = [NORMAL_FILENAME, NORMAL_FILENAME, NORMAL_FILENAME]
        #     if item.lm_num_diffs is not None and item.lm_num_diffs > 0:
        #         result[0] = CHANGED_FILENAME
        #         result[1] = CHANGED_FILENAME
        #     if item.mr_num_diffs is not None and item.mr_num_diffs > 0:
        #         result[1] = CHANGED_FILENAME
        #         result[2] = CHANGED_FILENAME
        #     return result
        # else:
        #     if item.middle is None:
        #         result = [INSERTED_FILENAME, ABSENT, INSERTED_FILENAME]
        #         if item.left is None:
        #             result[0] = ABSENT
        #         elif item.right is None:
        #             result[2] = ABSENT
        #         return result

        #     if (item.middle is not None and item.left is None
        #          and item.right is None):
        #         return [ABSENT, REMOVED_FILENAME, ABSENT]

        #     if item.middle is not None and count == 2:
        #         if item.left is None:
        #             if item.mr_num_diffs == 0:
        #                 return [ABSENT, NORMAL_FILENAME, NORMAL_FILENAME]
        #             else:
        #                 return [ABSENT, CHANGED_FILENAME, CHANGED_FILENAME]
        #         else:
        #             item.right is None
        #             if item.lm_num_diffs == 0:
        #                 return [NORMAL_FILENAME, NORMAL_FILENAME, ABSENT]
        #             else:
        #                 return [CHANGED_FILENAME, CHANGED_FILENAME, ABSENT]

        #     fixme: catch-all for now
        #     return [ERROR, ERROR, ERROR]

    def __render_line(self, row, col, text, color):
        self.canvas.draw_text(row, col, text, color)

    def __render_empty_item(self, row):
        out_string = (' ' * self.left_item_width + '|'
                      + ' ' * self.right_item_width)
        self.__render_line(row, 0, out_string, NORMAL_VERTICAL_SEP)

    def __next_display_line(self, path_const):
        candidate_path = self.__next_display_line_aux(path_const)
        if candidate_path is None:
            # print('None')
            return None
        # print('path=', candidate_path[0].left)

        # while candidate_path[0].IsHidden():
        while not self.__should_render_item(candidate_path[0]):
            candidate_path = self.__next_display_line_aux(candidate_path)
            # print('path=', candidate_path)
            if candidate_path is None:
                return None

        return candidate_path

    def __next_display_line_aux(self, path_const):
        path = path_const[:]
        current = path[0]

        if not current.collapse:
            # If the current item is not a leaf node, then the first child
            # is "next".
            if len(current.children) > 0:
                path.insert(0, current.children[0])
                return path

        # Otherwise, we're a leaf node, so the next is the sibling
        # that follows us in our parent's list of children. If we are
        # our parent's last sibling, we try the the node that follows
        # our parent in the grandparent's list, and so on. If at any
        # time we try to move up the tree and the ancestor is None (we
        # are at the last item in path[]), return None, since we must
        # be the last item in the tree.
        if len(path) == 1:
            # We're the root node, so there's no next.
            return None
        parent = path[1]

        while True:
            next = parent.find_following_child(current)
            if next is not None:
                path[0] = next
                return path
            else:
                path.pop(0)
                if len(path) == 1:
                    return None
                current = path[0]
                parent = path[1]

    def __prev_display_line(self, path_const):
        candidate_path = self.__prev_display_line_aux(path_const)
        if candidate_path is None:
            return None

        # while candidate_path[0].IsHidden():
        while not self.__should_render_item(candidate_path[0]):
            candidate_path = self.__prev_display_line_aux(candidate_path)
            if candidate_path is None:
                return None

        return candidate_path

    def __prev_display_line_aux(self, path_const):
        # If we are at the root node, just return None
        # if len(path_const) == 1:
        #    return None

        if (len(path_const) == 2
                and path_const[0] == self.model.top.children[0]):
            return None

        path = path_const[:]
        current = path[0]
        parent = path[1]

        prev = parent.find_previous_child(current)
        if prev is None:
            path.pop(0)
        else:
            current = prev
            path[0] = prev

            while True:
                if current.collapse:
                    break
                number_children = len(current.children)
                if number_children > 0:
                    current = current.children[number_children - 1]
                    path.insert(0, current)
                else:
                    break

        return path

    def __init_color_pairs(self):
        self.__init_a_color_pair(NORMAL_DIR_ARROW,
                                 'normal_dir_arrow_fg',
                                 'normal_dir_arrow_bg')
        self.__init_a_color_pair(NORMAL_FILENAME,
                                 'normal_filename_fg',
                                 'normal_filename_bg')
        self.__init_a_color_pair(NORMAL_COUNT,
                                 'normal_count_fg',
                                 'normal_count_bg')
        self.__init_a_color_pair(NORMAL_VERTICAL_SEP,
                                 'normal_vertical_sep_fg',
                                 'normal_vertical_sep_bg')
        self.__init_a_color_pair(NORMAL_HIDDEN,
                                 'normal_hidden_fg',
                                 'normal_hidden_bg')

        self.__init_a_color_pair(NORMAL_DIR_ARROW_H,
                                 'normal_dir_arrow_h_fg',
                                 'normal_dir_arrow_h_bg')
        self.__init_a_color_pair(NORMAL_FILENAME_H,
                                 'normal_filename_h_fg',
                                 'normal_filename_h_bg')
        self.__init_a_color_pair(NORMAL_COUNT_H,
                                 'normal_count_h_fg',
                                 'normal_count_h_bg')
        self.__init_a_color_pair(NORMAL_VERTICAL_SEP_H,
                                 'normal_vertical_sep_h_fg',
                                 'normal_vertical_sep_h_bg')
        self.__init_a_color_pair(NORMAL_HIDDEN_H,
                                 'normal_hidden_h_fg',
                                 'normal_hidden_h_bg')

        self.__init_a_color_pair(CHANGED_DIR_ARROW,
                                 'changed_dir_arrow_fg',
                                 'changed_dir_arrow_bg')
        self.__init_a_color_pair(CHANGED_FILENAME,
                                 'changed_filename_fg',
                                 'changed_filename_bg')
        self.__init_a_color_pair(CHANGED_COUNT,
                                 'changed_count_fg',
                                 'changed_count_bg')
        self.__init_a_color_pair(CHANGED_VERTICAL_SEP,
                                 'changed_vertical_sep_fg',
                                 'changed_vertical_sep_bg')
        self.__init_a_color_pair(CHANGED_HIDDEN,
                                 'changed_hidden_fg',
                                 'changed_hidden_bg')

        self.__init_a_color_pair(CHANGED_DIR_ARROW_H,
                                 'changed_dir_arrow_h_fg',
                                 'changed_dir_arrow_h_bg')
        self.__init_a_color_pair(CHANGED_FILENAME_H,
                                 'changed_filename_h_fg',
                                 'changed_filename_h_bg')
        self.__init_a_color_pair(CHANGED_COUNT_H,
                                 'changed_count_h_fg',
                                 'changed_count_h_bg')
        self.__init_a_color_pair(CHANGED_VERTICAL_SEP_H,
                                 'changed_vertical_sep_h_fg',
                                 'changed_vertical_sep_h_bg')
        self.__init_a_color_pair(CHANGED_HIDDEN_H,
                                 'changed_hidden_h_fg',
                                 'changed_hidden_h_bg')

        self.__init_a_color_pair(INSERTED_DIR_ARROW,
                                 'inserted_dir_arrow_fg',
                                 'inserted_dir_arrow_bg')
        self.__init_a_color_pair(INSERTED_FILENAME,
                                 'inserted_filename_fg',
                                 'inserted_filename_bg')
        self.__init_a_color_pair(INSERTED_COUNT,
                                 'inserted_count_fg',
                                 'inserted_count_bg')
        self.__init_a_color_pair(INSERTED_VERTICAL_SEP,
                                 'inserted_vertical_sep_fg',
                                 'inserted_vertical_sep_bg')
        self.__init_a_color_pair(INSERTED_HIDDEN,
                                 'inserted_hidden_fg',
                                 'inserted_hidden_bg')

        self.__init_a_color_pair(INSERTED_DIR_ARROW_H,
                                 'inserted_dir_arrow_h_fg',
                                 'inserted_dir_arrow_h_bg')
        self.__init_a_color_pair(INSERTED_FILENAME_H,
                                 'inserted_filename_h_fg',
                                 'inserted_filename_h_bg')
        self.__init_a_color_pair(INSERTED_COUNT_H,
                                 'inserted_count_h_fg',
                                 'inserted_count_h_bg')
        self.__init_a_color_pair(INSERTED_VERTICAL_SEP_H,
                                 'inserted_vertical_sep_h_fg',
                                 'inserted_vertical_sep_h_bg')
        self.__init_a_color_pair(INSERTED_HIDDEN_H,
                                 'inserted_hidden_h_fg',
                                 'inserted_hidden_h_bg')

        self.__init_a_color_pair(REMOVED_DIR_ARROW,
                                 'removed_dir_arrow_fg',
                                 'removed_dir_arrow_bg')
        self.__init_a_color_pair(REMOVED_FILENAME,
                                 'removed_filename_fg',
                                 'removed_filename_bg')
        self.__init_a_color_pair(REMOVED_COUNT,
                                 'removed_count_fg',
                                 'removed_count_bg')
        self.__init_a_color_pair(REMOVED_VERTICAL_SEP,
                                 'removed_vertical_sep_fg',
                                 'removed_vertical_sep_bg')
        self.__init_a_color_pair(REMOVED_HIDDEN,
                                 'removed_hidden_fg',
                                 'removed_hidden_bg')

        self.__init_a_color_pair(REMOVED_DIR_ARROW_H,
                                 'removed_dir_arrow_h_fg',
                                 'removed_dir_arrow_h_bg')
        self.__init_a_color_pair(REMOVED_FILENAME_H,
                                 'removed_filename_h_fg',
                                 'removed_filename_h_bg')
        self.__init_a_color_pair(REMOVED_COUNT_H,
                                 'removed_count_h_fg',
                                 'removed_count_h_bg')
        self.__init_a_color_pair(REMOVED_VERTICAL_SEP_H,
                                 'removed_vertical_sep_h_fg',
                                 'removed_vertical_sep_h_bg')
        self.__init_a_color_pair(REMOVED_HIDDEN_H,
                                 'removed_hidden_h_fg',
                                 'removed_hidden_h_bg')

        self.__init_a_color_pair(ABSENT,
                                 'absent_fg',
                                 'absent_bg')
        self.__init_a_color_pair(ABSENT_H,
                                 'absent_h_fg',
                                 'absent_h_bg')
        self.__init_a_color_pair(UNCOMPARED,
                                 'uncompared_fg',
                                 'uncompared_bg')
        self.__init_a_color_pair(UNCOMPARED_H,
                                 'uncompared_h_fg',
                                 'uncompared_h_bg')
        self.__init_a_color_pair(ERROR,
                                 'error_fg',
                                 'error_bg')
        self.__init_a_color_pair(ERROR_H,
                                 'error_h_fg',
                                 'error_h_bg')
        self.__init_a_color_pair(SELECTED,
                                 'selected_fg',
                                 'selected_bg')
        self.__init_a_color_pair(SELECTED_H,
                                 'selected_h_fg',
                                 'selected_h_bg')

        self.__init_a_color_pair(MARKER_OK,
                                 'marker_ok_fg',
                                 'marker_ok_bg')
        self.__init_a_color_pair(MARKER_MERGED,
                                 'marker_merged_fg',
                                 'marker_merged_bg')
        self.__init_a_color_pair(MARKER_RESOLVED,
                                 'marker_resolved_fg',
                                 'marker_resolved_bg')
        self.__init_a_color_pair(MARKER_CONFLICT,
                                 'marker_conflict_fg',
                                 'marker_conflict_bg')

        self.__init_a_color_pair(TOP_LINE,
                                 'top_line_fg',
                                 'top_line_bg')
        self.__init_a_color_pair(TOP_VERTICAL_SEP,
                                 'top_vertical_sep_fg',
                                 'top_vertical_sep_bg')

        self.__init_a_color_pair(STATUS_1,
                                 'status_1_fg',
                                 'status_1_bg')
        self.__init_a_color_pair(STATUS_2,
                                 'status_2_fg',
                                 'status_2_bg')
        self.__init_a_color_pair(STATUS_3,
                                 'status_3_fg',
                                 'status_3_bg')
        self.__init_a_color_pair(STATUS_4,
                                 'status_4_fg',
                                 'status_4_bg')

        self.__init_a_color_pair(PROMPT_1,
                                 'prompt_1_fg',
                                 'prompt_1_bg')
        self.__init_a_color_pair(PROMPT_2,
                                 'prompt_2_fg',
                                 'prompt_2_bg')
        self.__init_a_color_pair(PROMPT_3,
                                 'prompt_3_fg',
                                 'prompt_3_bg')
        self.__init_a_color_pair(PROMPT_4,
                                 'prompt_4_fg',
                                 'prompt_4_bg')
