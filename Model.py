# Model.py

import Match
import os
import subprocess
import threading
import time


NORMAL = 1
MISMATCH = 2
CONFLICT = 3

# Current state of the model
STATE_NORMAL = 0
STATE_NEW = 1
STATE_ENUMERATING = 2
STATE_PRECOMPARING = 3
STATE_COMPARING = 4
STATE_COPYING = 5
STATE_DELETING = 6


class Model:

    def __init__(self, left, right):
        self.left = left
        self.right = right
        self.modelstate = STATE_NEW

        self.tree_structure_lock = threading.Lock()

        self.operation_thread = None
        self.operation_arg = None

        self.comparison_thread = None
        self.stop_comparison = False

        self.top = Match.Match(left, right, None)
        self.top.mark_as_unenumerated()

    def destroy(self):
        # fixme: Kill any threads here.
        pass

    def state(self):
        return self.modelstate

    #
    # Operation thread
    #

    def request_operation(self, operation, item):

        # An operation is currently in progress, so signal an error
        # and do nothing.
        if self.operation_thread:
            return False

        # If this operation was requested for an item that has
        # descendants that are still uncompared, signal an error and
        # do nothing.
        if item.has_uncompared_descendants():
            print("has uncompared descendants. Doing nothing...")
            return False

        if operation == "refresh":
            # Enumeration can only take place if at least one of left
            # or right is present. Signal an error if neither is
            # present and remove the item from its parent.
            if self.__validate_item(item):
                self.__initiate_enumerate(item)
            else:
                # The left and right were both not present.  If this
                # is model.top, there is nothing we can do but give an
                # error.  If it is not the top, have the parent of
                # item remove item from its children.

                # FIXME: should this be self.top?  PyLint flags this as
                # an undefined variable.  It may be vestigial code that
                # should be eliminated.  Leave it until I figure out
                # what was going on.
                # if item is not model.top:
                #     item.parent.remove_child_from_children(item)
                return False
        elif operation == "copy_l2r":
            self.__initiate_copy_l2r(item)
        elif operation == "copy_r2l":
            self.__initiate_copy_r2l(item)
        elif operation == "delete":
            self.__initiate_delete(item)

    def __validate_item(self, item):
        print("in __validate_item()")
        return os.path.exists(item.left) or os.path.exists(item.right)

    def __initiate_enumerate(self, item):
        print("in __initiate_enumerate()")
        self.modelstate = STATE_ENUMERATING
        self.operation_thread = threading.Thread(target=self.__enumerate_aux)
        self.operation_arg = item
        self.operation_thread.start()

    def __enumerate_aux(self):
        with self.lock():
            self.operation_arg.enumerate()
            self.operation_arg.set_state_of_tree(Match.UNCOMPARED)
            self.operation_arg.num_diffs = 0
        self.modelstate = STATE_PRECOMPARING
        self.operation_thread = None  # Is this the right place to do this?
        self.operation_arg = None

        try:
            self.stop_comparison = True
            print("Stopping thread...")
            # print("thread=", self.comparison_thread)
            # print("is_alive()=", self.comparison_thread.is_alive())
            if self.comparison_thread:
                self.comparison_thread.join(1.0)
            # print("thread=", self.comparison_thread)
            # print("is_alive()=", self.comparison_thread.is_alive())
        except Exception:
            pass

        self.__initiate_compare()

    #
    # Comparison
    #
    def __initiate_compare(self):
        self.modelstate = STATE_COMPARING
        self.stop_comparison = False
        print("Creating comparison thread...")
        self.comparison_thread = threading.Thread(target=self.__compare_aux)
        print("new thread:", self.comparison_thread)
        self.comparison_thread.start()

    def __compare_aux(self):
        self.compare_match_item(self.top)
        self.modelstate = STATE_NORMAL
        # self.comparison_thread = None # Is this the right place to do this?

    def compare_match_item(self, item):
        if self.stop_comparison:
            return
        if item.state == Match.UNCOMPARED:
            item.compare_myself()
            time.sleep(0.001)  # fixme: just for testing.
        for child in item.children:
            # if self.stop_comparison:
            #     break
            # else:
            #     self.compare_match_item(child)
            self.compare_match_item(child)

    def lock(self):
        return self.tree_structure_lock

    #
    # Operations
    #
    def __initiate_copy_l2r(self, item):
        print("in __initiate_copy_l2r()")

        print("left :", item.left_pathname())
        print("right:", item.right_pathname())

        self.modelstate = STATE_COPYING
        self.operation_thread = threading.Thread(target=self.__copy_l2r_aux)
        self.operation_arg = item
        self.operation_thread.start()

    def __copy_l2r_aux(self):
        item = self.operation_arg

        if item.left is None:
            return

        left_name = item.left_pathname()
        right_name = item.right_pathname()

        if os.path.exists(right_name) and os.path.isdir(right_name):
            p1 = subprocess.Popen(["rm", "-Rf", right_name])
            p1.wait()
            if p1.returncode != 0:
                # Signal an error: fixme
                pass

        # Probably need to catch any exceptions here
        # Also, we need a loop here watching a variable to know if we
        # have been asked to cancel the operation.  Mark it as error
        # and then just return cleanly.  User terminate() or kill()
        # method on p1.
        p1 = subprocess.Popen(["cp", "-R", left_name, right_name])
        p1.wait()

        if p1.returncode == 0:
            # For now we can just start a refresh on the top item.  It
            # might be more efficient to just alter the items, eventually.
            item.right = right_name
            self.operation_thread = None
            self.request_operation("refresh", item)
        else:
            item.set_state_of_tree(Match.ERROR)
            self.modelstate = STATE_NORMAL
            self.operation_thread = None

    def __initiate_copy_r2l(self, item):
        print("in __initiate_copy_r2l()")

        print("left :", item.left_pathname())
        print("right:", item.right_pathname())

        self.modelstate = STATE_COPYING
        self.operation_thread = threading.Thread(target=self.__copy_r2l_aux)
        self.operation_arg = item
        self.operation_thread.start()

    def __copy_r2l_aux(self):
        item = self.operation_arg

        if item.right is None:
            return

        left_name = item.left_pathname()
        right_name = item.right_pathname()

        if os.path.exists(left_name) and os.path.isdir(left_name):
            p1 = subprocess.Popen(["rm", "-Rf", left_name])
            p1.wait()
            if p1.returncode != 0:
                # Signal an error: fixme
                pass

        # Probably need to catch any exceptions here
        # Also, we need a loop here watching a variable to know if we
        # have been asked to cancel the operation.  Mark it as error
        # and then just return cleanly.  User terminate() or kill()
        # method on p1.
        p1 = subprocess.Popen(["cp", "-R", right_name, left_name])
        p1.wait()

        if p1.returncode == 0:
            # For now we can just start a refresh on the top item.  It
            # might be more efficient to just alter the items, eventually.
            item.left = left_name
            self.operation_thread = None
            self.request_operation("refresh", item)
        else:
            item.set_state_of_tree(Match.ERROR)
            self.modelstate = STATE_NORMAL
            self.operation_thread = None

    def __initiate_delete(self, item):
        print("Deleting: ", item.left)

        exit_status_left = 0
        if item.left is not None:
            p1 = subprocess.Popen(["rm", "-Rf", item.left])
            p1.wait()
            exit_status_left = p1.returncode

        exit_status_right = 0
        if item.right is not None:
            p1 = subprocess.Popen(["rm", "-Rf", item.right])
            p1.wait()
            exit_status_right = p1.returncode

        # If either failed, we need to mark the item red.
        if exit_status_left != 0 or exit_status_right != 0:
            item.set_state_of_tree(Match.ERROR)
            return False  # don't need to reset cursor
        else:
            item.set_state_of_tree(Match.DELETED)
            return True  # need to reset cursor
